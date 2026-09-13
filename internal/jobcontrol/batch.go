package jobcontrol

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const batchPageSize = 50

// BatchStatus is the observable state of a resumable batch job, including
// enough information for clients to offer resuming a run that was interrupted
// by a restart.
type BatchStatus struct {
	Total         int
	Count         int
	Done          bool
	Running       bool
	Resumable     bool
	ConfigChanged bool
	Error         string
}

// PageLister is the subset of the query layer the batch job needs. It allows
// tests to drive pagination without a database.
type PageLister interface {
	ListTorrentsPageAfter(ctx context.Context, arg db.ListTorrentsPageAfterParams) ([]db.ListTorrentsPageAfterRow, error)
	CountTorrentsBefore(ctx context.Context, barrierTime pgtype.Timestamptz) (int64, error)
}

// BatchFunc processes one page of torrents and returns the number of items
// that count towards progress. Returning an error aborts the run.
type BatchFunc func(ctx context.Context, rows []db.ListTorrentsPageAfterRow) (int, error)

// StartOptions carries the per-run inputs of BatchTracker.Start.
type StartOptions struct {
	Queries     PageLister
	Fingerprint string
	ForceFresh  bool

	// PrepareFresh runs before a fresh (or forced) start, e.g. to recreate a
	// search index. It is not called when resuming.
	PrepareFresh func(ctx context.Context) error

	Run      BatchFunc
	OnFinish func()
	Logger   *zap.SugaredLogger
}

// BatchProgress tracks the mutable state of a running batch job. Its methods
// are safe for concurrent use.
type BatchProgress struct {
	mu               sync.Mutex
	total            int
	count            int
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

// Snapshot returns an immutable view of the current progress.
func (p *BatchProgress) Snapshot() BatchStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	return BatchStatus{
		Total:         p.total,
		Count:         p.count,
		Done:          p.done,
		Running:       p.running,
		Resumable:     p.resumable && !p.done && !p.running,
		ConfigChanged: p.configChanged,
		Error:         p.errMsg,
	}
}

func (p *BatchProgress) setRunning() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.running = true
}

// CanResume reports whether the tracked run has a barrier and has not
// finished. Runs interrupted before the barrier was established cannot be
// resumed.
func (p *BatchProgress) CanResume() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return !p.done && p.barrierTime.Valid && !p.barrierTime.Time.IsZero()
}

func (p *BatchProgress) fail(errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.done = false
	p.running = false
	p.errMsg = errMsg
}

// BatchTracker runs a resumable keyset-paginated job over the torrent table.
// It owns the shared job slot, persists progress after every page and can
// continue from the last processed torrent after a restart.
type BatchTracker struct {
	mu       sync.Mutex
	progress *BatchProgress
	ctrl     *Controller
	store    StateStore
	name     string
	key      string
	job      string
	label    string
}

// NewBatchTracker creates a tracker. name is used in log messages and errors
// (e.g. "reclassify"), key is the persisted state key, job is the
// Controller job name, and label names the progress counter (e.g. "processed"
// or "indexed").
func NewBatchTracker(name, key, job, label string, ctrl *Controller, store StateStore) *BatchTracker {
	return &BatchTracker{
		ctrl:  ctrl,
		store: store,
		name:  name,
		key:   key,
		job:   job,
		label: label,
	}
}

// Load restores persisted progress. It is called once during startup.
// currentFingerprint is compared against the stored fingerprint so a config
// change can be surfaced to the user without blocking the resume.
func (t *BatchTracker) Load(ctx context.Context, currentFingerprint string, logger *zap.SugaredLogger) error {
	if t.store == nil {
		return nil
	}

	state, ok, err := t.store.Load(ctx, t.key)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress = &BatchProgress{
		total:            state.Total,
		count:            state.Count,
		done:             state.Done,
		resumable:        !state.Done && !state.BarrierTime.IsZero(),
		configChanged:    state.Fingerprint != currentFingerprint,
		barrierTime:      timeToPgtype(state.BarrierTime),
		cursorCreatedAt:  timeToPgtype(state.CursorCreatedAt),
		cursorInfoHash:   state.CursorInfoHash,
		stateFingerprint: state.Fingerprint,
	}

	if logger != nil && !state.Done {
		logger.Infow("restored "+t.name+" progress",
			t.label, state.Count,
			"total", state.Total,
			"configChanged", t.progress.configChanged,
		)
	}

	return nil
}

// Start launches the job in the background. It fails fast when another job is
// running or this tracker already has one in progress.
func (t *BatchTracker) Start(ctx context.Context, opts StartOptions) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress != nil {
		status := t.progress.Snapshot()
		if status.Running && !status.Done {
			return fmt.Errorf("%s already in progress", t.name)
		}
	}

	if t.ctrl != nil && !t.ctrl.TryAcquire(t.job) {
		return fmt.Errorf("another operation in progress: %s", t.ctrl.Reason())
	}

	release := func() {
		if t.ctrl != nil {
			t.ctrl.Release(t.job)
		}
	}

	var progress *BatchProgress

	if !opts.ForceFresh && t.progress != nil && t.progress.CanResume() {
		progress = t.progress

		progress.mu.Lock()
		progress.configChanged = progress.stateFingerprint != opts.Fingerprint
		progress.mu.Unlock()
	} else {
		if opts.PrepareFresh != nil {
			if err := opts.PrepareFresh(ctx); err != nil {
				release()

				return err
			}
		}

		progress = &BatchProgress{
			resumable:        true,
			stateFingerprint: opts.Fingerprint,
		}
		t.progress = progress
	}

	// Mark the run as started before launching the goroutine so an immediate
	// failure still leaves a consistent status and concurrent callers are
	// rejected right away.
	progress.mu.Lock()
	progress.running = true
	progress.errMsg = ""
	progress.mu.Unlock()

	go t.run(ctx, opts, progress, release)

	return nil
}

// Progress returns the current status. A tracker that never ran reports Done.
func (t *BatchTracker) Progress() BatchStatus {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return BatchStatus{Done: true}
	}

	return t.progress.Snapshot()
}

// Discard drops the persisted progress and clears the tracker. It is called
// when the user chooses not to resume a pending run; the run will not be
// offered for resume again until a new one starts.
func (t *BatchTracker) Discard(ctx context.Context) error {
	t.mu.Lock()
	if t.progress != nil {
		status := t.progress.Snapshot()
		if status.Running && !status.Done {
			t.mu.Unlock()

			return fmt.Errorf("%s in progress", t.name)
		}
	}

	t.progress = nil
	t.mu.Unlock()

	if t.store == nil {
		return nil
	}

	return t.store.Delete(ctx, t.key)
}

// Running reports whether the job is currently in progress. It is used by
// other components (e.g. the DHT crawler) to pause work that would otherwise
// compete with the job.
func (t *BatchTracker) Running() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return false
	}

	status := t.progress.Snapshot()

	return status.Running && !status.Done
}

func (t *BatchTracker) run(ctx context.Context, opts StartOptions, progress *BatchProgress, release func()) {
	defer release()

	if opts.OnFinish != nil {
		defer opts.OnFinish()
	}

	progress.mu.Lock()
	resuming := progress.barrierTime.Valid && !progress.barrierTime.Time.IsZero()
	progress.mu.Unlock()

	save := func() {
		if t.store == nil {
			return
		}

		progress.mu.Lock()
		state := State{
			BarrierTime:     progress.barrierTime.Time,
			CursorCreatedAt: progress.cursorCreatedAt.Time,
			CursorInfoHash:  progress.cursorInfoHash,
			Total:           progress.total,
			Count:           progress.count,
			Done:            progress.done,
			Fingerprint:     progress.stateFingerprint,
		}
		progress.mu.Unlock()

		saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		if err := t.store.Save(saveCtx, t.key, state); err != nil {
			opts.Logger.Warnw("failed to save "+t.name+" progress", "error", err)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			progress.fail(fmt.Sprintf("%s panic: %v", t.name, r))
			save()
			opts.Logger.Errorw(t.name+" panicked", "panic", r)
		}
	}()

	var barrierTime pgtype.Timestamptz

	if resuming {
		progress.mu.Lock()
		barrierTime = progress.barrierTime
		progress.mu.Unlock()

		if total, err := opts.Queries.CountTorrentsBefore(ctx, barrierTime); err == nil {
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
		progress.count = 0
		progress.cursorCreatedAt = pgtype.Timestamptz{}
		progress.cursorInfoHash = ""
		progress.mu.Unlock()

		total, err := opts.Queries.CountTorrentsBefore(ctx, barrierTime)
		if err != nil {
			progress.fail("failed to count torrents: " + err.Error())
			save()

			return
		}

		progress.mu.Lock()
		progress.total = int(total)
		progress.mu.Unlock()
	}

	lastProgressLog := time.Now()

	for {
		if ctx.Err() != nil {
			progress.fail(t.name + " canceled: " + ctx.Err().Error())
			save()

			return
		}

		progress.mu.Lock()
		cursorCreatedAt := progress.cursorCreatedAt
		cursorInfoHash := progress.cursorInfoHash
		progress.mu.Unlock()

		rows, err := opts.Queries.ListTorrentsPageAfter(ctx, db.ListTorrentsPageAfterParams{
			BarrierTime:     barrierTime,
			CursorCreatedAt: cursorParam(cursorCreatedAt),
			CursorInfoHash:  cursorInfoHash,
			BatchSize:       batchPageSize,
		})
		if err != nil {
			progress.fail("failed to list torrents: " + err.Error())
			save()
			opts.Logger.Errorw(t.name+" failed", "error", err)

			return
		}

		if len(rows) == 0 {
			break
		}

		processed, err := opts.Run(ctx, rows)
		if err != nil {
			progress.fail(err.Error())
			save()
			opts.Logger.Errorw(t.name+" failed", "error", err)

			return
		}

		last := rows[len(rows)-1]

		progress.mu.Lock()
		progress.count += processed
		progress.cursorCreatedAt = last.Torrent.CreatedAt
		progress.cursorInfoHash = last.Torrent.InfoHash
		currentCount := progress.count
		currentTotal := progress.total
		progress.mu.Unlock()

		save()

		if currentCount >= currentTotal || time.Since(lastProgressLog) >= 30*time.Second {
			lastProgressLog = time.Now()

			opts.Logger.Infow(t.name+" progress", t.label, currentCount, "total", currentTotal)
		}

		if len(rows) < batchPageSize {
			break
		}
	}

	progress.mu.Lock()
	progress.done = true
	progress.resumable = false
	progress.mu.Unlock()

	save()
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
