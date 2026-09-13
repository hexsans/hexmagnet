package processor

import (
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/jobcontrol"
	"github.com/hexsans/hexmagnet/internal/protocol"
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

// ReclassifyTracker coordinates full-library reclassification. It reuses the
// regular processor pipeline (classify -> persist -> enrich) but walks every
// torrent in batches with ClassifyModeRematch, and holds the shared job slot so
// the crawler is paused and reindex cannot run concurrently.
//
// Progress is persisted through a StateStore so an interrupted run can be
// resumed from the last processed torrent after a restart.
type ReclassifyTracker struct {
	batch *jobcontrol.BatchTracker
}

func NewReclassifyTracker(ctrl *jobcontrol.Controller, store jobcontrol.StateStore) *ReclassifyTracker {
	return &ReclassifyTracker{
		batch: jobcontrol.NewBatchTracker(
			"reclassify",
			jobcontrol.KeyReclassifyState,
			jobcontrol.JobReclassify,
			"processed",
			ctrl,
			store,
		),
	}
}

// Load restores persisted progress. It is called once during startup.
// currentFingerprint is compared against the stored fingerprint so a config
// change can be surfaced to the user without blocking the resume.
func (t *ReclassifyTracker) Load(ctx context.Context, currentFingerprint string, logger *zap.SugaredLogger) error {
	return t.batch.Load(ctx, currentFingerprint, logger)
}

func (t *ReclassifyTracker) Start(
	ctx context.Context,
	proc Processor,
	queries *db.Queries,
	fingerprint string,
	logger *zap.SugaredLogger,
) error {
	return t.batch.Start(ctx, jobcontrol.StartOptions{
		Queries:     queries,
		Fingerprint: fingerprint,
		Run: func(ctx context.Context, rows []db.ListTorrentsPageAfterRow) (int, error) {
			hashes := make([]protocol.ID, 0, len(rows))
			for _, raw := range rows {
				hashes = append(hashes, db.ToProtocolID(raw.Torrent.InfoHash))
			}

			// Strict mode makes LLM failures abort the batch instead of
			// falling back to rules, so the run pauses and the operator can
			// decide whether to retry or cancel.
			if err := proc.Process(classifier.WithStrictLLM(ctx), MessageParams{
				InfoHashes:   hashes,
				ClassifyMode: ClassifyModeRematch,
			}); err != nil {
				return 0, fmt.Errorf("reclassify batch: %w", err)
			}

			return len(hashes), nil
		},
		Logger: logger,
	})
}

func (t *ReclassifyTracker) Progress() ReclassifyStatus {
	status := t.batch.Progress()

	return ReclassifyStatus{
		Total:         status.Total,
		Processed:     status.Count,
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
func (t *ReclassifyTracker) Discard(ctx context.Context) error {
	return t.batch.Discard(ctx)
}
