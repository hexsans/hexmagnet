package retryqueue

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// retryKey mirrors the composite primary key of torrent_retry_queue.
type retryKey struct {
	infoHash string
	stage    string
}

type fakeStore struct {
	mutex      sync.Mutex
	rows       map[retryKey]db.TorrentRetryQueue
	deleted    []string
	evicted    []string
	upsertErr  error
	listDueErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{rows: make(map[retryKey]db.TorrentRetryQueue)}
}

func (f *fakeStore) UpsertTorrentRetryBatch(
	_ context.Context,
	arg db.UpsertTorrentRetryBatchParams,
) ([]db.UpsertTorrentRetryBatchRow, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.upsertErr != nil {
		return nil, f.upsertErr
	}

	rows := make([]db.UpsertTorrentRetryBatchRow, 0, len(arg.InfoHashes))

	for i, hash := range arg.InfoHashes {
		key := retryKey{infoHash: hash, stage: arg.Stages[i]}

		// Emulates the SQL upsert: the failure count is incremented atomically.
		count := int32(1)
		if existing, ok := f.rows[key]; ok {
			count = existing.FailCount + 1
		}

		row := db.TorrentRetryQueue{
			InfoHash:    hash,
			Stage:       arg.Stages[i],
			Payload:     arg.Payloads[i],
			FailCount:   count,
			LastError:   arg.LastErrors[i],
			NextRetryAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			// dispatched_at is cleared: a new failure invalidates the old lease.
		}
		f.rows[key] = row

		rows = append(rows, db.UpsertTorrentRetryBatchRow{
			InfoHash:  hash,
			Stage:     arg.Stages[i],
			FailCount: count,
		})
	}

	return rows, nil
}

func (f *fakeStore) BackoffTorrentRetryBatch(_ context.Context, arg db.BackoffTorrentRetryBatchParams) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for i, hash := range arg.InfoHashes {
		key := retryKey{infoHash: hash, stage: arg.Stages[i]}

		row, ok := f.rows[key]
		if !ok {
			continue
		}

		row.NextRetryAt = pgtype.Timestamptz{
			Time:  time.Now().Add(time.Duration(arg.DelaySecs[i]) * time.Second),
			Valid: true,
		}
		f.rows[key] = row
	}

	return nil
}

func (f *fakeStore) DeleteTorrentRetry(_ context.Context, infoHash string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for key := range f.rows {
		if key.infoHash == infoHash {
			delete(f.rows, key)
		}
	}

	f.deleted = append(f.deleted, infoHash)

	return nil
}

func (f *fakeStore) ListDueTorrentRetry(_ context.Context, limit int32) ([]db.TorrentRetryQueue, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.listDueErr != nil {
		return nil, f.listDueErr
	}

	rows := make([]db.TorrentRetryQueue, 0)
	for _, row := range f.rows {
		if len(rows) >= int(limit) {
			break
		}

		if row.NextRetryAt.Valid && !row.NextRetryAt.Time.After(time.Now()) {
			rows = append(rows, row)
		}
	}

	return rows, nil
}

func (f *fakeStore) MarkTorrentRetryDispatched(_ context.Context, arg db.MarkTorrentRetryDispatchedParams) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	key := retryKey{infoHash: arg.InfoHash, stage: arg.Stage}

	row, ok := f.rows[key]
	if !ok {
		return nil
	}

	row.NextRetryAt = arg.NextRetryAt
	row.FailCount = arg.FailCount
	row.DispatchedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	f.rows[key] = row

	return nil
}

func (f *fakeStore) TouchTorrentRetry(_ context.Context, infoHash string) (int64, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	touched := int64(0)

	for key, row := range f.rows {
		if key.infoHash != infoHash {
			continue
		}

		row.NextRetryAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		row.DispatchedAt = pgtype.Timestamptz{}
		f.rows[key] = row
		touched++
	}

	return touched, nil
}

func (f *fakeStore) ListTorrentRetryQueue(_ context.Context, _ db.ListTorrentRetryQueueParams) ([]db.TorrentRetryQueue, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	rows := make([]db.TorrentRetryQueue, 0, len(f.rows))
	for _, row := range f.rows {
		rows = append(rows, row)
	}

	return rows, nil
}

func (f *fakeStore) CountTorrentRetryQueue(context.Context) (int64, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return int64(len(f.rows)), nil
}

func (f *fakeStore) ClearTorrentRetryQueue(context.Context) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.rows = make(map[retryKey]db.TorrentRetryQueue)

	return nil
}

func (f *fakeStore) DeleteTorrent(_ context.Context, infoHash string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.evicted = append(f.evicted, infoHash)

	return nil
}

type fakeProducer struct {
	mutex    sync.Mutex
	messages []fakeMessage
}

type fakeMessage struct {
	topic string
	key   string
	value any
}

func (p *fakeProducer) Produce(topic string, key string, value any) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.messages = append(p.messages, fakeMessage{topic: topic, key: key, value: value})
}

func (*fakeProducer) Close() error { return nil }

func newTestQueue(_ *testing.T, store *fakeStore, producer *fakeProducer, cfg Config) *Queue {
	q := &Queue{
		queries:       utils.NewLazy(func() (*db.Queries, error) { return nil, errors.New("not used in tests") }),
		producer:      producer,
		storeOverride: store,
		logger:        zap.NewNop().Sugar(),
	}
	q.cfg.Set(cfg)

	return q
}

func testConfig() Config {
	return Config{
		MaxRetries:    3,
		Interval:      time.Minute,
		BackoffFactor: 2,
		MaxInterval:   time.Hour,
	}
}

func TestBackoffDelay(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Interval:      time.Minute,
		BackoffFactor: 2,
		MaxInterval:   10 * time.Minute,
	}

	assert.Equal(t, time.Minute, cfg.backoffDelay(1))
	assert.Equal(t, 2*time.Minute, cfg.backoffDelay(2))
	assert.Equal(t, 4*time.Minute, cfg.backoffDelay(3))
	assert.Equal(t, 8*time.Minute, cfg.backoffDelay(4))
	assert.Equal(t, 10*time.Minute, cfg.backoffDelay(5))
	assert.Equal(t, 10*time.Minute, cfg.backoffDelay(100))
}

func TestBackoffDelay_Defaults(t *testing.T) {
	t.Parallel()

	cfg := Config{}

	assert.Equal(t, 5*time.Minute, cfg.backoffDelay(1))
	assert.Equal(t, 5*time.Minute, cfg.backoffDelay(100))
}

func TestEnqueue_NewEntry(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}
	q := newTestQueue(t, store, producer, testConfig())

	err := q.Enqueue(context.Background(), StageClassify, "hash-a", map[string]any{"k": "v"}, errors.New("tmdb down"))
	require.NoError(t, err)

	row, ok := store.rows[retryKey{infoHash: "hash-a", stage: string(StageClassify)}]
	require.True(t, ok)
	assert.Equal(t, int32(1), row.FailCount)
	assert.Contains(t, string(row.Payload), `"k"`)
	assert.WithinDuration(t, time.Now().Add(time.Minute), row.NextRetryAt.Time, 5*time.Second)
	assert.Empty(t, producer.messages)
}

func TestEnqueue_IncrementsFailCount(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}
	q := newTestQueue(t, store, producer, testConfig())

	require.NoError(t, q.Enqueue(context.Background(), StageEnrich, "hash-a", []string{"id-1"}, errors.New("boom")))
	require.NoError(t, q.Enqueue(context.Background(), StageEnrich, "hash-a", []string{"id-1"}, errors.New("boom")))

	assert.EqualValues(t, 2, store.rows[retryKey{infoHash: "hash-a", stage: string(StageEnrich)}].FailCount)
}

func TestEnqueue_StagesAreIsolated(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}
	q := newTestQueue(t, store, producer, testConfig())

	require.NoError(t, q.Enqueue(context.Background(), StageClassify, "hash-a", map[string]any{"mode": "classify"}, errors.New("boom")))
	require.NoError(t, q.Enqueue(context.Background(), StageEnrich, "hash-a", []string{"id-a"}, errors.New("boom")))
	require.NoError(t, q.Enqueue(context.Background(), StageClassify, "hash-a", map[string]any{"mode": "classify"}, errors.New("boom")))

	assert.EqualValues(t, 2, store.rows[retryKey{infoHash: "hash-a", stage: string(StageClassify)}].FailCount)
	assert.EqualValues(t, 1, store.rows[retryKey{infoHash: "hash-a", stage: string(StageEnrich)}].FailCount)
	assert.Len(t, store.rows, 2)
}

func TestEnqueue_ExceededDeletesTorrent(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}

	cfg := testConfig()
	cfg.MaxRetries = 2
	q := newTestQueue(t, store, producer, cfg)

	require.NoError(t, q.Enqueue(context.Background(), StageClassify, "hash-a", nil, errors.New("boom")))
	require.NoError(t, q.Enqueue(context.Background(), StageClassify, "hash-a", nil, errors.New("boom")))

	err := q.Enqueue(context.Background(), StageClassify, "hash-a", nil, errors.New("boom"))
	require.ErrorIs(t, err, ErrExceeded)

	assert.Empty(t, store.rows)
	assert.Contains(t, store.evicted, "hash-a")
	assert.Contains(t, store.deleted, "hash-a")

	require.Len(t, producer.messages, 1)
	assert.Equal(t, kafka.TopicDeleteTorrent, producer.messages[0].topic)
	assert.Equal(t, "hash-a", producer.messages[0].value)
}

func TestEnqueue_NegativeMaxRetriesAlwaysRetries(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}

	cfg := testConfig()
	cfg.MaxRetries = -1
	q := newTestQueue(t, store, producer, cfg)

	key := retryKey{infoHash: "hash-a", stage: string(StageClassify)}
	store.rows[key] = db.TorrentRetryQueue{
		InfoHash:  "hash-a",
		Stage:     string(StageClassify),
		FailCount: 1000,
	}

	err := q.Enqueue(context.Background(), StageClassify, "hash-a", nil, errors.New("boom"))
	require.NoError(t, err)
	assert.EqualValues(t, 1001, store.rows[key].FailCount)
}

func TestEnqueueBatch(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}
	q := newTestQueue(t, store, producer, testConfig())

	items := []RetryItem{
		{Stage: StageClassify, InfoHash: "hash-a", Payload: map[string]any{"i": 1}},
		{Stage: StageClassify, InfoHash: "hash-b", Payload: map[string]any{"i": 2}},
	}

	require.NoError(t, q.EnqueueBatch(context.Background(), items, errors.New("boom")))
	assert.EqualValues(t, 1, store.rows[retryKey{infoHash: "hash-a", stage: string(StageClassify)}].FailCount)
	assert.EqualValues(t, 1, store.rows[retryKey{infoHash: "hash-b", stage: string(StageClassify)}].FailCount)

	// Empty batches are a no-op.
	require.NoError(t, q.EnqueueBatch(context.Background(), nil, errors.New("boom")))
}

func TestRemove(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}
	q := newTestQueue(t, store, producer, testConfig())

	store.rows[retryKey{infoHash: "hash-a", stage: string(StageClassify)}] = db.TorrentRetryQueue{
		InfoHash: "hash-a", Stage: string(StageClassify), FailCount: 1,
	}
	store.rows[retryKey{infoHash: "hash-a", stage: string(StageEnrich)}] = db.TorrentRetryQueue{
		InfoHash: "hash-a", Stage: string(StageEnrich), FailCount: 1,
	}

	q.Remove(context.Background(), "hash-a", "hash-missing")

	assert.Empty(t, store.rows)
}

func TestDispatchDue(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}

	cfg := testConfig()
	cfg.DispatchLease = 10 * time.Minute
	q := newTestQueue(t, store, producer, cfg)

	require.NoError(t, q.Enqueue(context.Background(), StageClassify, "hash-a", map[string]any{"x": 1}, errors.New("boom")))
	require.NoError(t, q.Enqueue(context.Background(), StageEnrich, "hash-b", []string{"id-b"}, errors.New("boom")))

	// Make everything due.
	for key, row := range store.rows {
		row.NextRetryAt = pgtype.Timestamptz{Time: time.Now().Add(-time.Minute), Valid: true}
		store.rows[key] = row
	}

	q.dispatchDue(context.Background())

	require.Len(t, producer.messages, 2)

	topics := map[string]fakeMessage{}
	for _, msg := range producer.messages {
		topics[msg.key] = msg
	}

	assert.Equal(t, kafka.TopicProcessTorrent, topics["hash-a"].topic)
	assert.Equal(t, kafka.TopicEnriched, topics["hash-b"].topic)

	// Payloads must be valid JSON, not base64-encoded strings.
	raw, ok := topics["hash-a"].value.(json.RawMessage)
	require.True(t, ok, "payload must be json.RawMessage, got %T", topics["hash-a"].value)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.InDelta(t, float64(1), decoded["x"], 0.0001)

	rawEnrich, ok := topics["hash-b"].value.(json.RawMessage)
	require.True(t, ok)

	var ids []string
	require.NoError(t, json.Unmarshal(rawEnrich, &ids))
	assert.Equal(t, []string{"id-b"}, ids)

	// Entries must be leased, not deleted, until the retry outcome is known.
	assert.Len(t, store.rows, 2)
	assert.True(t, store.rows[retryKey{infoHash: "hash-a", stage: string(StageClassify)}].NextRetryAt.Time.After(time.Now()))
	assert.True(t, store.rows[retryKey{infoHash: "hash-a", stage: string(StageClassify)}].DispatchedAt.Valid)
}

func TestDispatchDue_LeaseExpiryCountsAsFailure(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}

	cfg := testConfig()
	cfg.MaxRetries = 1
	cfg.DispatchLease = 10 * time.Minute
	q := newTestQueue(t, store, producer, cfg)

	key := retryKey{infoHash: "hash-a", stage: string(StageClassify)}

	// First failure: enqueued (fail_count = 1) and dispatched.
	require.NoError(t, q.Enqueue(context.Background(), StageClassify, "hash-a", nil, errors.New("boom")))

	row := store.rows[key]
	row.NextRetryAt = pgtype.Timestamptz{Time: time.Now().Add(-time.Minute), Valid: true}
	store.rows[key] = row

	q.dispatchDue(context.Background())

	dispatched := store.rows[key]
	assert.EqualValues(t, 1, dispatched.FailCount)
	assert.True(t, dispatched.DispatchedAt.Valid)

	// The dispatch lease expired without an outcome being reported: the next
	// scan must count the lost attempt against the budget and evict the entry.
	expired := store.rows[key]
	expired.NextRetryAt = pgtype.Timestamptz{Time: time.Now().Add(-time.Minute), Valid: true}
	store.rows[key] = expired

	q.dispatchDue(context.Background())

	assert.Empty(t, store.rows)
	assert.Contains(t, store.evicted, "hash-a")
}

func TestUpdateConfig(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}
	q := newTestQueue(t, store, producer, testConfig())

	q.UpdateConfig(Config{MaxRetries: 1})

	assert.Equal(t, 1, q.cfg.Get().MaxRetries)
}

func TestRetryNow(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	producer := &fakeProducer{}
	q := newTestQueue(t, store, producer, testConfig())

	store.rows[retryKey{infoHash: "hash-a", stage: string(StageClassify)}] = db.TorrentRetryQueue{
		InfoHash: "hash-a", Stage: string(StageClassify), FailCount: 1,
	}

	require.NoError(t, q.RetryNow(context.Background(), "hash-a"))

	row := store.rows[retryKey{infoHash: "hash-a", stage: string(StageClassify)}]
	assert.WithinDuration(t, time.Now(), row.NextRetryAt.Time, 5*time.Second)

	// An unknown hash must be an error, not a silent success.
	require.ErrorIs(t, q.RetryNow(context.Background(), "hash-missing"), pgx.ErrNoRows)
}
