package indexer

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/jobcontrol"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func RecreateIndex(ctx context.Context, es *elasticsearch.Client, dims int, logger *zap.SugaredLogger) error {
	if err := es.DeleteIndex(ctx, IndexName); err != nil {
		logger.Warnw("delete index (may not exist)", "index", IndexName, "error", err)
	}

	if err := es.CreateIndex(ctx, IndexName, bytes.NewReader([]byte(IndexMapping(dims)))); err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	logger.Infow("recreated index", "index", IndexName, "dims", dims)

	return nil
}

// ReindexStatus is the observable state of a full-library reindex, including
// enough information for clients to offer resuming a run that was interrupted
// by a restart.
type ReindexStatus struct {
	Total         int
	Indexed       int
	Done          bool
	Running       bool
	Resumable     bool
	ConfigChanged bool
	Error         string
}

type ReindexProgress struct {
	mu               sync.Mutex
	total            int
	indexed          int
	done             bool
	errMsg           string
	running          bool
	resumable        bool
	configChanged    bool
	barrierTime      pgtype.Timestamptz
	cursorCreatedAt  pgtype.Timestamptz
	cursorInfoHash   string
	stateFingerprint string
}

func (p *ReindexProgress) Snapshot() ReindexStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	return ReindexStatus{
		Total:         p.total,
		Indexed:       p.indexed,
		Done:          p.done,
		Running:       p.running,
		Resumable:     p.resumable && !p.done && !p.running,
		ConfigChanged: p.configChanged,
		Error:         p.errMsg,
	}
}

func (p *ReindexProgress) setRunning() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.running = true
}

// canResume reports whether the tracked run has a barrier and has not finished.
// Runs interrupted before the barrier was established cannot be resumed.
func (p *ReindexProgress) canResume() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return !p.done && p.barrierTime.Valid && !p.barrierTime.Time.IsZero()
}

func (p *ReindexProgress) fail(errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.done = false
	p.running = false
	p.errMsg = errMsg
}

type ReindexTracker struct {
	mu       sync.Mutex
	progress *ReindexProgress
	ctrl     *jobcontrol.Controller
	store    jobcontrol.StateStore
}

func NewReindexTracker(ctrl *jobcontrol.Controller, store jobcontrol.StateStore) *ReindexTracker {
	return &ReindexTracker{ctrl: ctrl, store: store}
}

// Load restores persisted progress. It is called once during startup.
// currentFingerprint is compared against the stored fingerprint so a config
// change can be surfaced to the user without blocking the resume.
func (t *ReindexTracker) Load(ctx context.Context, currentFingerprint string, logger *zap.SugaredLogger) error {
	if t.store == nil {
		return nil
	}

	state, ok, err := t.store.Load(ctx, jobcontrol.KeyReindexState)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress = &ReindexProgress{
		total:            state.Total,
		indexed:          state.Count,
		done:             state.Done,
		resumable:        !state.Done && !state.BarrierTime.IsZero(),
		configChanged:    state.Fingerprint != currentFingerprint,
		barrierTime:      timeToPgtype(state.BarrierTime),
		cursorCreatedAt:  timeToPgtype(state.CursorCreatedAt),
		cursorInfoHash:   state.CursorInfoHash,
		stateFingerprint: state.Fingerprint,
	}

	if logger != nil && !state.Done {
		logger.Infow("restored reindex progress",
			"indexed", state.Count,
			"total", state.Total,
			"configChanged", t.progress.configChanged,
		)
	}

	return nil
}

//nolint:revive // Start needs the full set of runtime dependencies plus resume options
func (t *ReindexTracker) Start(
	ctx context.Context,
	es *elasticsearch.Client,
	embedder *embedding.Client,
	queries *db.Queries,
	dims int,
	maxSearchFiles int,
	fingerprint string,
	forceFresh bool,
	logger *zap.SugaredLogger,
) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress != nil {
		status := t.progress.Snapshot()
		if status.Running && !status.Done {
			return fmt.Errorf("reindex already in progress")
		}
	}

	if t.ctrl != nil && !t.ctrl.TryAcquire(jobcontrol.JobReindex) {
		return fmt.Errorf("another operation in progress: %s", t.ctrl.Reason())
	}

	release := func() {
		if t.ctrl != nil {
			t.ctrl.Release(jobcontrol.JobReindex)
		}
	}

	var progress *ReindexProgress

	if !forceFresh && t.progress != nil && t.progress.canResume() {
		progress = t.progress

		progress.mu.Lock()
		progress.running = false
		progress.errMsg = ""
		progress.configChanged = progress.stateFingerprint != fingerprint
		progress.mu.Unlock()
	} else {
		if err := RecreateIndex(ctx, es, dims, logger); err != nil {
			release()

			return fmt.Errorf("failed to recreate index: %w", err)
		}

		progress = &ReindexProgress{
			resumable:        true,
			stateFingerprint: fingerprint,
		}
		t.progress = progress
	}

	go runReindex(ctx, queries, es, embedder, maxSearchFiles, progress, t.store, release, logger)

	waitForReindexStart(progress)

	return nil
}

func (t *ReindexTracker) Progress() ReindexStatus {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return ReindexStatus{Done: true}
	}

	return t.progress.Snapshot()
}

// Discard drops the persisted progress and clears the tracker. It is called
// when the user chooses not to resume a pending run; the run will not be
// offered for resume again until a new one starts.
func (t *ReindexTracker) Discard(ctx context.Context) error {
	t.mu.Lock()
	if t.progress != nil {
		status := t.progress.Snapshot()
		if status.Running && !status.Done {
			t.mu.Unlock()

			return fmt.Errorf("reindex in progress")
		}
	}

	t.progress = nil
	t.mu.Unlock()

	if t.store == nil {
		return nil
	}

	return t.store.Delete(ctx, jobcontrol.KeyReindexState)
}

// Running reports whether a reindex is currently in progress. It is used by
// other components (e.g. the DHT crawler) to pause work that would otherwise
// compete with the reindex.
func (t *ReindexTracker) Running() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return false
	}

	status := t.progress.Snapshot()

	return status.Running && !status.Done
}

//nolint:revive // runReindex threads the full set of runtime dependencies
func runReindex(
	ctx context.Context,
	queries *db.Queries,
	es *elasticsearch.Client,
	embedder *embedding.Client,
	maxSearchFiles int,
	progress *ReindexProgress,
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
			Count:           progress.indexed,
			Done:            progress.done,
			Fingerprint:     progress.stateFingerprint,
		}
		progress.mu.Unlock()

		saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		if err := store.Save(saveCtx, jobcontrol.KeyReindexState, state); err != nil {
			logger.Warnw("failed to save reindex progress", "error", err)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			progress.fail(fmt.Sprintf("reindex panic: %v", r))
			save()
			logger.Errorw("reindex panicked", "panic", r)
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
		progress.indexed = 0
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

	limiter := rate.NewLimiter(rate.Limit(50), 50)

	skippedFiles := 0

	lastProgressLog := time.Now()

	for {
		if ctx.Err() != nil {
			progress.fail("reindex canceled: " + ctx.Err().Error())
			save()

			return
		}

		progress.mu.Lock()
		cursorCreatedAt := progress.cursorCreatedAt
		cursorInfoHash := progress.cursorInfoHash
		progress.mu.Unlock()

		torrents, err := queries.ListTorrentsPageAfter(ctx, db.ListTorrentsPageAfterParams{
			BarrierTime:     barrierTime,
			CursorCreatedAt: cursorParam(cursorCreatedAt),
			CursorInfoHash:  cursorInfoHash,
			BatchSize:       batchSize,
		})
		if err != nil {
			progress.fail("failed to list torrents: " + err.Error())
			save()
			logger.Errorw("reindex failed", "error", err)

			return
		}

		if len(torrents) == 0 {
			break
		}

		docs := make([]TorrentContentDocument, 0, len(torrents))

		for _, raw := range torrents {
			t := db.TorrentToModel(raw.Torrent, raw.Seeders, raw.Leechers)

			files, err := queries.ListTorrentFiles(ctx, raw.Torrent.InfoHash)
			if err != nil {
				skippedFiles++

				logger.Debugw("failed to list torrent files", "info_hash", raw.Torrent.InfoHash, "error", err)

				continue
			}

			for i := range files {
				t.Files = append(t.Files, db.TorrentFileToModel(files[i]))
			}

			doc := NewDocument(t)
			docs = append(docs, doc)
		}

		if len(docs) > 0 && embedder != nil {
			texts := make([]string, len(docs))
			for i := range docs {
				texts[i] = BuildSearchText(docs[i], maxSearchFiles)
			}

			vectors, embedErr := embedder.Embed(ctx, texts)
			if embedErr != nil {
				logger.Warnw("reindex embedding failed, continuing without vectors", "error", embedErr)
			} else {
				for i := range docs {
					docs[i].SearchVector = vectors[i]
				}
			}
		}

		if len(docs) > 0 {
			body, err := BulkBody(docs)
			if err != nil {
				progress.fail("failed to build bulk body: " + err.Error())
				save()
				logger.Errorw("reindex failed", "error", err)

				return
			}

			if err := limiter.Wait(ctx); err != nil {
				progress.fail("reindex canceled: " + err.Error())
				save()

				return
			}

			if err := es.BulkIndex(ctx, bytes.NewReader(body)); err != nil {
				logger.Warnw("reindex bulk index failed for batch, continuing", "error", err, "cursor", cursorInfoHash)
			}
		}

		last := torrents[len(torrents)-1]

		progress.mu.Lock()
		progress.indexed += len(docs)
		progress.cursorCreatedAt = last.Torrent.CreatedAt
		progress.cursorInfoHash = last.Torrent.InfoHash
		currentIndexed := progress.indexed
		currentTotal := progress.total
		progress.mu.Unlock()

		save()

		if currentIndexed >= currentTotal || time.Since(lastProgressLog) >= 30*time.Second {
			lastProgressLog = time.Now()

			logger.Infow("reindex progress", "indexed", currentIndexed, "total", currentTotal)
		}

		if len(torrents) < batchSize {
			break
		}
	}

	if skippedFiles > 0 {
		logger.Warnw("reindex skipped torrents", "skipped", skippedFiles)
	}

	progress.mu.Lock()
	progress.done = true
	progress.resumable = false
	progress.mu.Unlock()

	save()
}

func waitForReindexStart(pos *ReindexProgress) {
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
