package retryqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Stage identifies which pipeline stage a retry entry belongs to.
type Stage string

const (
	StageClassify Stage = "classify"
	StageEnrich   Stage = "enrich"
)

// ErrExceeded is returned by Enqueue when a torrent exceeded the configured
// retry budget and has been deleted instead of re-enqueued.
var ErrExceeded = errors.New("retry count exceeded")

// errDispatchExpired is reported as the cause when an entry becomes due again
// while still carrying a dispatch lease, meaning the previous retry outcome
// was never reported.
var errDispatchExpired = errors.New("dispatch lease expired without outcome")

// Store is the persistence surface used by the retry queue. It is satisfied
// by *db.Queries.
type Store interface {
	UpsertTorrentRetryBatch(ctx context.Context, arg db.UpsertTorrentRetryBatchParams) ([]db.UpsertTorrentRetryBatchRow, error)
	BackoffTorrentRetryBatch(ctx context.Context, arg db.BackoffTorrentRetryBatchParams) error
	DeleteTorrentRetry(ctx context.Context, infoHash string) error
	ListDueTorrentRetry(ctx context.Context, limit int32) ([]db.TorrentRetryQueue, error)
	MarkTorrentRetryDispatched(ctx context.Context, arg db.MarkTorrentRetryDispatchedParams) error
	TouchTorrentRetry(ctx context.Context, infoHash string) (int64, error)
	ListTorrentRetryQueue(ctx context.Context, arg db.ListTorrentRetryQueueParams) ([]db.TorrentRetryQueue, error)
	CountTorrentRetryQueue(ctx context.Context) (int64, error)
	ClearTorrentRetryQueue(ctx context.Context) error
	DeleteTorrent(ctx context.Context, infoHash string) error
}

// Entry is a read-model view of a retry queue row.
type Entry struct {
	InfoHash      string
	Stage         string
	FailCount     int
	LastError     string
	LastFailureAt time.Time
	NextRetryAt   time.Time
}

// RetryItem is a single torrent to record in the retry queue.
type RetryItem struct {
	Stage    Stage
	InfoHash string
	Payload  any
}

type Queue struct {
	queries       utils.Lazy[*db.Queries]
	storeOverride Store // test hook, bypasses the queries lazy
	producer      queue.Producer
	cfg           concurrency.AtomicValue[Config]
	logger        *zap.SugaredLogger
}

type QueueParams struct {
	fx.In
	Queries  utils.Lazy[*db.Queries]
	Producer queue.Producer
	Logger   *zap.SugaredLogger
	Cfg      Config
}

func NewQueue(p QueueParams) *Queue {
	cfg := p.Cfg
	if cfg == (Config{}) {
		cfg = NewDefaultConfig()
	}

	q := &Queue{
		queries:  p.Queries,
		producer: p.Producer,
		logger:   p.Logger.Named("retry_queue"),
	}
	q.cfg.Set(cfg)

	return q
}

// UpdateConfig hot-swaps the retry queue configuration.
func (q *Queue) UpdateConfig(cfg Config) {
	q.cfg.Set(cfg)
}

func (q *Queue) Enabled() bool {
	return q.cfg.Get().Enabled
}

func (q *Queue) store() (Store, error) {
	if q.storeOverride != nil {
		return q.storeOverride, nil
	}

	queries, err := q.queries.Get()
	if err != nil {
		return nil, err
	}

	return queries, nil
}

// Enqueue records a failure for a single torrent.
func (q *Queue) Enqueue(ctx context.Context, stage Stage, infoHash string, payload any, cause error) error {
	return q.EnqueueBatch(ctx, []RetryItem{
		{Stage: stage, InfoHash: infoHash, Payload: payload},
	}, cause)
}

// EnqueueBatch records failures for a batch of torrents in a single upsert.
// The failure count is incremented atomically per (info_hash, stage); when the
// count exceeds the configured budget the torrent is deleted instead and
// ErrExceeded is returned. The payload is the message that will be re-published
// on dispatch.
func (q *Queue) EnqueueBatch(ctx context.Context, items []RetryItem, cause error) error {
	if len(items) == 0 {
		return nil
	}

	cfg := q.cfg.Get()

	st, err := q.store()
	if err != nil {
		return err
	}

	hashes := make([]string, 0, len(items))
	stages := make([]string, 0, len(items))
	payloads := make([][]byte, 0, len(items))
	lastErrors := make([]string, 0, len(items))

	lastErr := ""
	if cause != nil {
		lastErr = cause.Error()
	}

	for _, item := range items {
		data, err := json.Marshal(item.Payload)
		if err != nil {
			return fmt.Errorf("marshal retry payload: %w", err)
		}

		hashes = append(hashes, item.InfoHash)
		stages = append(stages, string(item.Stage))
		payloads = append(payloads, data)
		lastErrors = append(lastErrors, lastErr)
	}

	rows, err := st.UpsertTorrentRetryBatch(ctx, db.UpsertTorrentRetryBatchParams{
		InfoHashes: hashes,
		Stages:     stages,
		Payloads:   payloads,
		LastErrors: lastErrors,
	})
	if err != nil {
		return fmt.Errorf("upsert retry entries: %w", err)
	}

	backoffHashes := make([]string, 0, len(rows))
	backoffStages := make([]string, 0, len(rows))
	backoffDelays := make([]int32, 0, len(rows))

	for _, row := range rows {
		if cfg.MaxRetries >= 0 && row.FailCount > int32(cfg.MaxRetries) {
			q.evict(ctx, st, row.InfoHash, row.FailCount, cause)

			continue
		}

		backoffHashes = append(backoffHashes, row.InfoHash)
		backoffStages = append(backoffStages, row.Stage)
		backoffDelays = append(backoffDelays, int32(cfg.backoffDelay(row.FailCount).Seconds()))

		q.logger.Warnw("torrent queued for retry",
			"stage", row.Stage,
			"info_hash", row.InfoHash,
			"fail_count", row.FailCount,
			"error", lastErr,
		)
	}

	if len(backoffHashes) > 0 {
		err := st.BackoffTorrentRetryBatch(ctx, db.BackoffTorrentRetryBatchParams{
			InfoHashes: backoffHashes,
			Stages:     backoffStages,
			DelaySecs:  backoffDelays,
		})
		if err != nil {
			// Non-fatal: entries keep their previous next_retry_at and stay
			// dispatchable.
			q.logger.Errorw("failed to apply retry backoff", "error", err)
		}
	}

	if len(backoffHashes) < len(rows) {
		return fmt.Errorf("%w for %d torrent(s)", ErrExceeded, len(rows)-len(backoffHashes))
	}

	return nil
}

// evict deletes the torrent after its retry budget was exhausted. The torrent
// is removed from the database and a delete event is published so the search
// backend can clean up. It is deliberately not blocked: it may re-enter the
// pipeline if it is discovered again.
func (q *Queue) evict(ctx context.Context, st Store, infoHash string, failCount int32, cause error) {
	q.logger.Warnw("retry budget exceeded, deleting torrent",
		"info_hash", infoHash,
		"fail_count", failCount,
		"error", cause,
	)

	if err := st.DeleteTorrent(ctx, infoHash); err != nil {
		q.logger.Errorw("failed to delete torrent after retry budget exceeded",
			"info_hash", infoHash,
			"error", err,
		)
	}

	q.producer.Produce(kafka.TopicDeleteTorrent, infoHash, infoHash)

	if err := st.DeleteTorrentRetry(ctx, infoHash); err != nil {
		q.logger.Errorw("failed to delete retry entry",
			"info_hash", infoHash,
			"error", err,
		)
	}
}

// Remove clears retry entries once processing succeeded. Best-effort: errors
// are logged and swallowed.
func (q *Queue) Remove(ctx context.Context, infoHashes ...string) {
	st, err := q.store()
	if err != nil {
		q.logger.Warnw("failed to resolve store, skipping retry cleanup", "error", err)

		return
	}

	for _, infoHash := range infoHashes {
		if err := st.DeleteTorrentRetry(ctx, infoHash); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			q.logger.Errorw("failed to remove retry entry",
				"info_hash", infoHash,
				"error", err,
			)
		}
	}
}

// dispatchDue publishes all due entries to their pipeline topics and marks
// them with a dispatch lease so they are not re-published before the outcome
// of the retry is known. An entry that becomes due again while still leased
// means the previous outcome was never reported; it is counted as a failed
// attempt before being re-dispatched.
func (q *Queue) dispatchDue(ctx context.Context) {
	cfg := q.cfg.Get()
	if !cfg.Enabled {
		return
	}

	st, err := q.store()
	if err != nil {
		q.logger.Errorw("failed to resolve store", "error", err)

		return
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	rows, err := st.ListDueTorrentRetry(ctx, int32(batchSize))
	if err != nil {
		q.logger.Errorw("failed to list due retry entries", "error", err)

		return
	}

	if len(rows) == 0 {
		return
	}

	lease := cfg.DispatchLease
	if lease <= 0 {
		lease = 15 * time.Minute
	}

	now := time.Now()

	for _, row := range rows {
		topic := kafka.TopicProcessTorrent
		if Stage(row.Stage) == StageEnrich {
			topic = kafka.TopicEnriched
		}

		failCount := row.FailCount
		cause := error(nil)

		if row.DispatchedAt.Valid {
			// The previous dispatch never reported an outcome: treat it as a
			// failed attempt so lost messages cannot bypass the retry budget.
			failCount = row.FailCount + 1
			cause = errDispatchExpired
		}

		if cfg.MaxRetries >= 0 && failCount > int32(cfg.MaxRetries) {
			q.evict(ctx, st, row.InfoHash, failCount, cause)

			continue
		}

		// The payload is stored as valid JSON in the database; json.RawMessage
		// keeps the producers from re-encoding the bytes as a base64 string.
		q.producer.Produce(topic, row.InfoHash, json.RawMessage(row.Payload))

		err := st.MarkTorrentRetryDispatched(ctx, db.MarkTorrentRetryDispatchedParams{
			InfoHash:    row.InfoHash,
			Stage:       row.Stage,
			FailCount:   failCount,
			NextRetryAt: pgtype.Timestamptz{Time: now.Add(lease), Valid: true},
		})
		if err != nil {
			q.logger.Errorw("failed to mark retry entry dispatched",
				"info_hash", row.InfoHash,
				"error", err,
			)
		} else {
			q.logger.Infow("dispatched retry entry",
				"stage", row.Stage,
				"info_hash", row.InfoHash,
				"fail_count", failCount,
			)
		}
	}
}

// Entries returns a paginated view of the retry queue for observability.
func (q *Queue) Entries(ctx context.Context, limit, offset int) ([]Entry, int64, error) {
	st, err := q.store()
	if err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}

	if offset < 0 {
		offset = 0
	}

	rows, err := st.ListTorrentRetryQueue(ctx, db.ListTorrentRetryQueueParams{
		Limit:  utils.ClampInt32(limit),
		Offset: utils.ClampInt32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list retry queue: %w", err)
	}

	total, err := st.CountTorrentRetryQueue(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count retry queue: %w", err)
	}

	entries := make([]Entry, 0, len(rows))
	for _, row := range rows {
		entry := Entry{
			InfoHash:  row.InfoHash,
			Stage:     row.Stage,
			FailCount: int(row.FailCount),
			LastError: row.LastError,
		}

		if row.LastFailureAt.Valid {
			entry.LastFailureAt = row.LastFailureAt.Time
		}

		if row.NextRetryAt.Valid {
			entry.NextRetryAt = row.NextRetryAt.Time
		}

		entries = append(entries, entry)
	}

	return entries, total, nil
}

// RetryNow makes the entry due for the next scheduler scan. It returns
// pgx.ErrNoRows when no entry exists for the given hash.
func (q *Queue) RetryNow(ctx context.Context, infoHash string) error {
	st, err := q.store()
	if err != nil {
		return err
	}

	rows, err := st.TouchTorrentRetry(ctx, infoHash)
	if err != nil {
		return err
	}

	if rows == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Clear removes every entry from the retry queue.
func (q *Queue) Clear(ctx context.Context) error {
	st, err := q.store()
	if err != nil {
		return err
	}

	return st.ClearTorrentRetryQueue(ctx)
}
