package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/jobcontrol"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// ReclassifyStatus is the observable state of a full-library reclassification
// run, including enough information for clients to offer resuming a run that
// was interrupted by a restart.
type ReclassifyStatus struct {
	Total         int
	Processed     int
	Done          bool
	Running       bool
	Resumable     bool
	ConfigChanged bool
	Error         string
}

// ReclassifyProgress tracks the state of a full-library reclassification run.
type ReclassifyProgress struct {
	mu               sync.Mutex
	total            int
	processed        int
	done             bool
	running          bool
	resumable        bool
	configChanged    bool
	errMsg           string
	barrierTime      pgtype.Timestamptz
	cursorCreatedAt  pgtype.Timestamptz
	cursorInfoHash   string
	stateFingerprint string
}

func (p *ReclassifyProgress) Snapshot() ReclassifyStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	return ReclassifyStatus{
		Total:         p.total,
		Processed:     p.processed,
		Done:          p.done,
		Running:       p.running,
		Resumable:     p.resumable && !p.done && !p.running,
		ConfigChanged: p.configChanged,
		Error:         p.errMsg,
	}
}

// canResume reports whether the tracked run has a barrier and has not finished.
// Runs interrupted before the barrier was established cannot be resumed.
func (p *ReclassifyProgress) canResume() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return !p.done && p.barrierTime.Valid && !p.barrierTime.Time.IsZero()
}

func (p *ReclassifyProgress) fail(errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.done = false
	p.running = false
	p.errMsg = errMsg
}

// ReclassifyTracker coordinates full-library reclassification. It reuses the
// regular processor pipeline (classify -> persist -> enrich) but walks every
// torrent in batches with ClassifyModeRematch, and holds the shared job slot so
// the crawler is paused and reindex cannot run concurrently.
//
// Progress is persisted through a StateStore so an interrupted run can be
// resumed from the last processed torrent after a restart.
type ReclassifyTracker struct {
	mu       sync.Mutex
	progress *ReclassifyProgress
	ctrl     *jobcontrol.Controller
	store    jobcontrol.StateStore
}

func NewReclassifyTracker(ctrl *jobcontrol.Controller, store jobcontrol.StateStore) *ReclassifyTracker {
	return &ReclassifyTracker{ctrl: ctrl, store: store}
}

// Load restores persisted progress. It is called once during startup.
// currentFingerprint is compared against the stored fingerprint so a config
// change can be surfaced to the user without blocking the resume.
func (t *ReclassifyTracker) Load(ctx context.Context, currentFingerprint string, logger *zap.SugaredLogger) error {
	if t.store == nil {
		return nil
	}

	state, ok, err := t.store.Load(ctx, jobcontrol.KeyReclassifyState)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress = &ReclassifyProgress{
		total:            state.Total,
		processed:        state.Count,
		done:             state.Done,
		resumable:        !state.Done && !state.BarrierTime.IsZero(),
		configChanged:    state.Fingerprint != currentFingerprint,
		errMsg:           "",
		barrierTime:      timeToPgtype(state.BarrierTime),
		cursorCreatedAt:  timeToPgtype(state.CursorCreatedAt),
		cursorInfoHash:   state.CursorInfoHash,
		stateFingerprint: state.Fingerprint,
	}

	if logger != nil && !state.Done {
		logger.Infow("restored reclassify progress",
			"processed", state.Count,
			"total", state.Total,
			"configChanged", t.progress.configChanged,
		)
	}

	return nil
}

func (t *ReclassifyTracker) Start(
	ctx context.Context,
	proc Processor,
	queries *db.Queries,
	fingerprint string,
	logger *zap.SugaredLogger,
) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress != nil {
		status := t.progress.Snapshot()
		if status.Running && !status.Done {
			return fmt.Errorf("reclassify already in progress")
		}
	}

	if t.ctrl != nil && !t.ctrl.TryAcquire(jobcontrol.JobReclassify) {
		return fmt.Errorf("another operation in progress: %s", t.ctrl.Reason())
	}

	release := func() {
		if t.ctrl != nil {
			t.ctrl.Release(jobcontrol.JobReclassify)
		}
	}

	var progress *ReclassifyProgress

	if t.progress != nil && t.progress.canResume() {
		progress = t.progress

		progress.mu.Lock()
		progress.running = false
		progress.errMsg = ""
		progress.configChanged = progress.stateFingerprint != fingerprint
		progress.mu.Unlock()
	} else {
		progress = &ReclassifyProgress{
			resumable:        true,
			stateFingerprint: fingerprint,
		}
		t.progress = progress
	}

	go runReclassify(ctx, proc, queries, progress, t.store, release, logger)

	waitForReclassifyStart(progress)

	return nil
}

func (t *ReclassifyTracker) Progress() ReclassifyStatus {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return ReclassifyStatus{Done: true}
	}

	return t.progress.Snapshot()
}

// Discard drops the persisted progress and clears the tracker. It is called
// when the user chooses not to resume a pending run; the run will not be
// offered for resume again until a new one starts.
func (t *ReclassifyTracker) Discard(ctx context.Context) error {
	t.mu.Lock()
	if t.progress != nil {
		status := t.progress.Snapshot()
		if status.Running && !status.Done {
			t.mu.Unlock()

			return fmt.Errorf("reclassify in progress")
		}
	}

	t.progress = nil
	t.mu.Unlock()

	if t.store == nil {
		return nil
	}

	return t.store.Delete(ctx, jobcontrol.KeyReclassifyState)
}

func runReclassify(
	ctx context.Context,
	proc Processor,
	queries *db.Queries,
	progress *ReclassifyProgress,
	store jobcontrol.StateStore,
	release func(),
	logger *zap.SugaredLogger,
) {
	defer release()

	progress.mu.Lock()
	resuming := progress.barrierTime.Valid && !progress.barrierTime.Time.IsZero()
	progress.running = true
	progress.mu.Unlock()

	save := func() {
		if store == nil {
			return
		}

		progress.mu.Lock()
		state := jobcontrol.State{
			BarrierTime:     progress.barrierTime.Time,
			CursorCreatedAt: progress.cursorCreatedAt.Time,
			CursorInfoHash:  progress.cursorInfoHash,
			Total:           progress.total,
			Count:           progress.processed,
			Done:            progress.done,
			Fingerprint:     progress.stateFingerprint,
		}
		progress.mu.Unlock()

		saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		if err := store.Save(saveCtx, jobcontrol.KeyReclassifyState, state); err != nil {
			logger.Warnw("failed to save reclassify progress", "error", err)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			progress.fail(fmt.Sprintf("reclassify panic: %v", r))
			save()
			logger.Errorw("reclassify panicked", "panic", r)
		}
	}()

	var barrierTime pgtype.Timestamptz

	if resuming {
		progress.mu.Lock()
		barrierTime = progress.barrierTime
		progress.mu.Unlock()

		if total, err := queries.CountTorrentsBefore(ctx, barrierTime); err == nil {
			progress.mu.Lock()
			progress.total = int(total)
			progress.mu.Unlock()
		}
	} else {
		barrierTime = pgtype.Timestamptz{
			Time:  time.Now().UTC(),
			Valid: true,
		}

		progress.mu.Lock()
		progress.barrierTime = barrierTime
		progress.total = 0
		progress.processed = 0
		progress.cursorCreatedAt = pgtype.Timestamptz{}
		progress.cursorInfoHash = ""
		progress.mu.Unlock()

		total, err := queries.CountTorrentsBefore(ctx, barrierTime)
		if err != nil {
			progress.fail("failed to count torrents: " + err.Error())
			save()

			return
		}

		progress.mu.Lock()
		progress.total = int(total)
		progress.mu.Unlock()
	}

	const batchSize = 50

	lastProgressLog := time.Now()

	for {
		if ctx.Err() != nil {
			progress.fail("reclassify canceled: " + ctx.Err().Error())
			save()

			return
		}

		progress.mu.Lock()
		cursorCreatedAt := progress.cursorCreatedAt
		cursorInfoHash := progress.cursorInfoHash
		progress.mu.Unlock()

		rows, err := queries.ListTorrentsPageAfter(ctx, db.ListTorrentsPageAfterParams{
			BarrierTime:     barrierTime,
			CursorCreatedAt: cursorParam(cursorCreatedAt),
			CursorInfoHash:  cursorInfoHash,
			BatchSize:       batchSize,
		})
		if err != nil {
			progress.fail("failed to list torrents: " + err.Error())
			save()
			logger.Errorw("reclassify failed", "error", err)

			return
		}

		if len(rows) == 0 {
			break
		}

		hashes := make([]protocol.ID, 0, len(rows))
		for _, raw := range rows {
			hashes = append(hashes, db.ToProtocolID(raw.Torrent.InfoHash))
		}

		if err := proc.Process(ctx, MessageParams{
			InfoHashes:   hashes,
			ClassifyMode: ClassifyModeRematch,
		}); err != nil {
			logger.Warnw("reclassify batch failed, continuing", "error", err, "cursor", cursorInfoHash)
		}

		last := rows[len(rows)-1]

		progress.mu.Lock()
		progress.processed += len(hashes)
		progress.cursorCreatedAt = last.Torrent.CreatedAt
		progress.cursorInfoHash = last.Torrent.InfoHash
		currentProcessed := progress.processed
		currentTotal := progress.total
		progress.mu.Unlock()

		save()

		if currentProcessed >= currentTotal || time.Since(lastProgressLog) >= 30*time.Second {
			lastProgressLog = time.Now()

			logger.Infow("reclassify progress", "processed", currentProcessed, "total", currentTotal)
		}

		if len(rows) < batchSize {
			break
		}
	}

	progress.mu.Lock()
	progress.done = true
	progress.resumable = false
	progress.mu.Unlock()

	save()
}

func waitForReclassifyStart(pos *ReclassifyProgress) {
	startedAt := time.Now()

	for {
		pos.mu.Lock()
		running := pos.running
		pos.mu.Unlock()

		if running {
			return
		}

		if time.Since(startedAt) > 5*time.Second {
			return
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func timeToPgtype(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: t, Valid: true}
}

// cursorParam replaces an empty cursor with the Unix epoch so keyset pagination
// starts at the beginning. Passing NULL would make the row comparison evaluate
// to NULL and return no rows.
func cursorParam(t pgtype.Timestamptz) pgtype.Timestamptz {
	if !t.Valid || t.Time.IsZero() {
		return pgtype.Timestamptz{Time: time.Unix(0, 0).UTC(), Valid: true}
	}

	return t
}
