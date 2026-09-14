package indexer

import (
	"bytes"
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/jobcontrol"
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

type ReindexTracker struct {
	batch *jobcontrol.BatchTracker
}

func NewReindexTracker(ctrl *jobcontrol.Controller, store jobcontrol.StateStore) *ReindexTracker {
	return &ReindexTracker{
		batch: jobcontrol.NewBatchTracker(
			"reindex",
			jobcontrol.KeyReindexState,
			jobcontrol.JobReindex,
			"indexed",
			ctrl,
			store,
		),
	}
}

// Load restores persisted progress. It is called once during startup.
// currentFingerprint is compared against the stored fingerprint so a config
// change can be surfaced to the user without blocking the resume.
func (t *ReindexTracker) Load(ctx context.Context, currentFingerprint string, logger *zap.SugaredLogger) error {
	return t.batch.Load(ctx, currentFingerprint, logger)
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
	limiter := rate.NewLimiter(rate.Limit(50), 50)

	return t.batch.Start(ctx, jobcontrol.StartOptions{
		Queries:     queries,
		Fingerprint: fingerprint,
		ForceFresh:  forceFresh,
		PrepareFresh: func(ctx context.Context) error {
			if err := RecreateIndex(ctx, es, dims, logger); err != nil {
				return fmt.Errorf("failed to recreate index: %w", err)
			}

			return nil
		},
		Run: func(ctx context.Context, torrents []db.ListTorrentsPageAfterRow) (int, error) {
			docs := make([]TorrentContentDocument, 0, len(torrents))

			for _, raw := range torrents {
				torrent, err := db.TorrentWithFiles(ctx, queries, raw.Torrent, raw.Seeders, raw.Leechers)
				if err != nil {
					return 0, fmt.Errorf("load torrent %s: %w", raw.Torrent.InfoHash, err)
				}

				docs = append(docs, NewDocument(torrent))
			}

			if len(docs) > 0 && embedder != nil {
				texts := make([]string, len(docs))
				for i := range docs {
					texts[i] = BuildSearchText(docs[i], maxSearchFiles)
				}

				vectors, embedErr := embedder.Embed(ctx, texts)
				if embedErr != nil {
					return 0, fmt.Errorf("embed batch of %d torrents: %w", len(docs), embedErr)
				}

				for i := range docs {
					docs[i].SearchVector = vectors[i]
				}
			}

			if len(docs) > 0 {
				body, err := BulkBody(docs)
				if err != nil {
					return 0, fmt.Errorf("failed to build bulk body: %w", err)
				}

				if err := limiter.Wait(ctx); err != nil {
					return 0, fmt.Errorf("reindex canceled: %w", err)
				}

				if err := es.BulkIndex(ctx, bytes.NewReader(body)); err != nil {
					return 0, fmt.Errorf("bulk index batch of %d torrents: %w", len(docs), err)
				}
			}

			return len(docs), nil
		},
		Logger: logger,
	})
}

func (t *ReindexTracker) Progress() ReindexStatus {
	status := t.batch.Progress()

	return ReindexStatus{
		Total:         status.Total,
		Indexed:       status.Count,
		Done:          status.Done,
		Running:       status.Running,
		Resumable:     status.Resumable,
		ConfigChanged: status.ConfigChanged,
		Error:         status.Error,
	}
}

// Discard drops the persisted progress and clears the tracker. It is called
// when the user chooses not to resume a pending run; the run will not be
// offered for resume again until a new one starts.
func (t *ReindexTracker) Discard(ctx context.Context) error {
	return t.batch.Discard(ctx)
}

// Running reports whether a reindex is currently in progress. It is used by
// other components (e.g. the DHT crawler) to pause work that would otherwise
// compete with the reindex.
func (t *ReindexTracker) Running() bool {
	return t.batch.Running()
}
