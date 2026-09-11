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

// ReclassifyProgress tracks the state of a full-library reclassification run.
type ReclassifyProgress struct {
	mu        sync.Mutex
	total     int
	processed int
	done      bool
	running   bool
	errMsg    string
}

func (p *ReclassifyProgress) Snapshot() (total, processed int, done, running bool, errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.total, p.processed, p.done, p.running, p.errMsg
}

func (p *ReclassifyProgress) setRunning() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.running = true
}

// ReclassifyTracker coordinates full-library reclassification. It reuses the
// regular processor pipeline (classify -> persist -> enrich) but walks every
// torrent in batches with ClassifyModeRematch, and holds the shared job slot so
// the crawler is paused and reindex cannot run concurrently.
type ReclassifyTracker struct {
	mu       sync.Mutex
	progress *ReclassifyProgress
	ctrl     *jobcontrol.Controller
}

func NewReclassifyTracker(ctrl *jobcontrol.Controller) *ReclassifyTracker {
	return &ReclassifyTracker{ctrl: ctrl}
}

func (t *ReclassifyTracker) Start(
	ctx context.Context,
	proc Processor,
	queries *db.Queries,
	logger *zap.SugaredLogger,
) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress != nil {
		_, _, done, _, _ := t.progress.Snapshot()
		if !done {
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

	t.progress = &ReclassifyProgress{}
	go runReclassify(ctx, proc, queries, t.progress, release, logger)

	waitForReclassifyStart(t.progress)

	return nil
}

func (t *ReclassifyTracker) Progress() (total, processed int, done, running bool, errMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return 0, 0, true, false, ""
	}

	return t.progress.Snapshot()
}

func runReclassify(
	ctx context.Context,
	proc Processor,
	queries *db.Queries,
	progress *ReclassifyProgress,
	release func(),
	logger *zap.SugaredLogger,
) {
	defer release()

	defer func() {
		if r := recover(); r != nil {
			progress.mu.Lock()
			progress.done = true
			progress.errMsg = fmt.Sprintf("reclassify panic: %v", r)
			progress.mu.Unlock()
			logger.Errorw("reclassify panicked", "panic", r)
		}
	}()

	progress.setRunning()

	barrierTime := pgtype.Timestamptz{
		Time:  time.Now().UTC(),
		Valid: true,
	}

	total, err := queries.CountTorrentsBefore(ctx, barrierTime)
	if err != nil {
		progress.mu.Lock()
		progress.done = true
		progress.errMsg = "failed to count torrents: " + err.Error()
		progress.mu.Unlock()

		return
	}

	progress.mu.Lock()
	progress.total = int(total)
	progress.mu.Unlock()

	const batchSize = 50

	offset := int32(0)

	for {
		if ctx.Err() != nil {
			progress.mu.Lock()
			progress.done = true
			progress.errMsg = "reclassify canceled: " + ctx.Err().Error()
			progress.mu.Unlock()

			return
		}

		rows, err := queries.ListTorrentsPaginatedBefore(ctx, db.ListTorrentsPaginatedBeforeParams{
			Column1: barrierTime,
			Limit:   batchSize,
			Offset:  offset,
		})
		if err != nil {
			progress.mu.Lock()
			progress.done = true
			progress.errMsg = "failed to list torrents: " + err.Error()
			progress.mu.Unlock()
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
			logger.Warnw("reclassify batch failed, continuing", "error", err, "offset", offset)
		}

		progress.mu.Lock()
		progress.processed += len(hashes)
		currentProcessed := progress.processed
		currentTotal := progress.total
		progress.mu.Unlock()

		logger.Infow("reclassify progress", "processed", currentProcessed, "total", currentTotal)

		offset += batchSize

		if len(rows) < batchSize {
			break
		}
	}

	progress.mu.Lock()
	progress.done = true
	progress.mu.Unlock()
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
