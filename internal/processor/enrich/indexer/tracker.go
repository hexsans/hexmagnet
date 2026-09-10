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

type ReindexProgress struct {
	mu      sync.Mutex
	total   int
	indexed int
	done    bool
	errMsg  string
	running bool
}

func (p *ReindexProgress) Snapshot() (total, indexed int, done bool, running bool, errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.total, p.indexed, p.done, p.running, p.errMsg
}

func (p *ReindexProgress) setRunning() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.running = true
}

type ReindexTracker struct {
	mu       sync.Mutex
	progress *ReindexProgress
}

func NewReindexTracker() *ReindexTracker {
	return &ReindexTracker{}
}

func (t *ReindexTracker) Start(
	ctx context.Context,
	es *elasticsearch.Client,
	embedder *embedding.Client,
	queries *db.Queries,
	dims int,
	maxSearchFiles int,
	logger *zap.SugaredLogger,
) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress != nil {
		_, _, done, _, _ := t.progress.Snapshot()
		if !done {
			return fmt.Errorf("reindex already in progress")
		}
	}

	if err := RecreateIndex(ctx, es, dims, logger); err != nil {
		return fmt.Errorf("failed to recreate index: %w", err)
	}

	t.progress = &ReindexProgress{}
	go runReindex(ctx, queries, es, embedder, maxSearchFiles, t.progress, logger)

	waitForReindexStart(t.progress)

	return nil
}

func (t *ReindexTracker) Progress() (total, indexed int, done bool, running bool, errMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress == nil {
		return 0, 0, true, false, ""
	}

	return t.progress.Snapshot()
}

func runReindex(
	ctx context.Context,
	queries *db.Queries,
	es *elasticsearch.Client,
	embedder *embedding.Client,
	maxSearchFiles int,
	progress *ReindexProgress,
	logger *zap.SugaredLogger,
) {
	defer func() {
		if r := recover(); r != nil {
			progress.mu.Lock()
			progress.done = true
			progress.errMsg = fmt.Sprintf("reindex panic: %v", r)
			progress.mu.Unlock()
			logger.Errorw("reindex panicked", "panic", r)
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

	limiter := rate.NewLimiter(rate.Limit(50), 50)

	offset := int32(0)

	for {
		if ctx.Err() != nil {
			progress.mu.Lock()
			progress.done = true
			progress.errMsg = "reindex canceled: " + ctx.Err().Error()
			progress.mu.Unlock()

			return
		}

		torrents, err := queries.ListTorrentsPaginatedBefore(ctx, db.ListTorrentsPaginatedBeforeParams{
			Column1: barrierTime,
			Limit:   batchSize,
			Offset:  offset,
		})
		if err != nil {
			progress.mu.Lock()
			progress.done = true
			progress.errMsg = "failed to list torrents: " + err.Error()
			progress.mu.Unlock()
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
				logger.Warnw("failed to list torrent files", "info_hash", raw.Torrent.InfoHash, "error", err)
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
				progress.mu.Lock()
				progress.done = true
				progress.errMsg = "failed to build bulk body: " + err.Error()
				progress.mu.Unlock()
				logger.Errorw("reindex failed", "error", err)

				return
			}

			if err := limiter.Wait(ctx); err != nil {
				return
			}

			if err := es.BulkIndex(ctx, bytes.NewReader(body)); err != nil {
				logger.Warnw("reindex bulk index failed for batch, continuing", "error", err, "offset", offset)
			}
		}

		progress.mu.Lock()
		progress.indexed += len(docs)
		currentIndexed := progress.indexed
		currentTotal := progress.total
		progress.mu.Unlock()

		logger.Infow("reindex progress", "indexed", currentIndexed, "total", currentTotal)

		offset += batchSize

		if len(torrents) < batchSize {
			break
		}
	}

	progress.mu.Lock()
	progress.done = true
	progress.mu.Unlock()
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
