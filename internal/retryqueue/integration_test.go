//go:build integration

package retryqueue

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newIntegrationStore connects to a real Postgres instance. The DSN is read
// from HEXMAGNET_TEST_DSN, e.g.
// postgres://postgres:postgres@localhost:5432/hexmagnet?sslmode=disable
func newIntegrationStore(t *testing.T) (Store, *pgx.Conn) {
	t.Helper()

	dsn := os.Getenv("HEXMAGNET_TEST_DSN")
	if dsn == "" {
		t.Skip("HEXMAGNET_TEST_DSN not set")
	}

	conn, err := pgx.Connect(context.Background(), dsn)
	require.NoError(t, err)

	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	_, err = conn.Exec(context.Background(), `TRUNCATE torrent_retry_queue`)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `TRUNCATE torrent_retry_queue`)
	})

	return db.NewQueriesWithDBTX(conn), conn
}

func newIntegrationQueue(t *testing.T) (*Queue, *pgx.Conn) {
	t.Helper()

	store, conn := newIntegrationStore(t)

	q := &Queue{
		producer:      &fakeProducer{},
		storeOverride: store,
		logger:        zap.NewNop().Sugar(),
	}
	q.cfg.Set(testConfig())

	return q, conn
}

func TestIntegration_EnqueueBatch_AtomicIncrement(t *testing.T) {
	q, conn := newIntegrationQueue(t)
	ctx := context.Background()

	items := []RetryItem{
		{Stage: StageClassify, InfoHash: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", Payload: map[string]any{"i": 1}},
		{Stage: StageEnrich, InfoHash: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", Payload: []string{"id-x"}},
	}

	require.NoError(t, q.EnqueueBatch(ctx, items, assert.AnError))

	key := retryKey{infoHash: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", stage: string(StageClassify)}

	// Same hash, different stages: isolated rows.
	require.NoError(t, q.EnqueueBatch(ctx, items[:1], assert.AnError))

	var failCount int32
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT fail_count FROM torrent_retry_queue WHERE info_hash = $1 AND stage = $2`,
		key.infoHash, key.stage).Scan(&failCount))
	assert.EqualValues(t, 2, failCount, "fail_count must increment atomically")

	require.NoError(t, conn.QueryRow(ctx,
		`SELECT fail_count FROM torrent_retry_queue WHERE info_hash = $1 AND stage = $2`,
		key.infoHash, string(StageEnrich)).Scan(&failCount))
	assert.EqualValues(t, 1, failCount, "stages must be isolated")

	// Backoff was applied after the upsert: fail_count = 2 with a factor of 2
	// yields a 2 minute delay.
	var nextRetryAt time.Time
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT next_retry_at FROM torrent_retry_queue WHERE info_hash = $1 AND stage = $2`,
		key.infoHash, key.stage).Scan(&nextRetryAt))
	assert.WithinDuration(t, time.Now().Add(2*time.Minute), nextRetryAt, 10*time.Second)
}

func TestIntegration_BudgetExceededEvicts(t *testing.T) {
	store, conn := newIntegrationStore(t)
	ctx := context.Background()

	cfg := testConfig()
	cfg.MaxRetries = 2
	cfg.Interval = time.Second
	cfg.DispatchLease = time.Second

	producer := &fakeProducer{}
	q := &Queue{producer: producer, storeOverride: store, logger: zap.NewNop().Sugar()}
	q.cfg.Set(cfg)

	hash := "b1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"

	require.NoError(t, q.Enqueue(ctx, StageClassify, hash, nil, assert.AnError))
	require.NoError(t, q.Enqueue(ctx, StageClassify, hash, nil, assert.AnError))
	require.ErrorIs(t, q.Enqueue(ctx, StageClassify, hash, nil, assert.AnError), ErrExceeded)

	var count int
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT COUNT(*) FROM torrent_retry_queue WHERE info_hash = $1`, hash).Scan(&count))
	assert.Equal(t, 0, count, "entry must be removed after the budget is exhausted")
}

func TestIntegration_DispatchDue_SendsRawJSON(t *testing.T) {
	q, conn := newIntegrationQueue(t)
	ctx := context.Background()

	cfg := testConfig()
	cfg.Interval = time.Second
	cfg.DispatchLease = time.Minute

	producer := &fakeProducer{}
	q.producer = producer
	c := q.cfg.Get()
	c.Interval = cfg.Interval
	c.DispatchLease = cfg.DispatchLease
	q.cfg.Set(c)

	payload := map[string]any{"InfoHashes": []string{"c1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"}, "ClassifyMode": "default"}
	require.NoError(t, q.Enqueue(ctx, StageClassify, "c1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", payload, assert.AnError))

	// Wait for the entry to become due.
	time.Sleep(1500 * time.Millisecond)

	q.dispatchDue(ctx)

	require.Len(t, producer.messages, 1)
	msg := producer.messages[0]
	assert.Equal(t, kafka.TopicProcessTorrent, msg.topic)

	raw, ok := msg.value.(json.RawMessage)
	require.True(t, ok, "payload must be json.RawMessage, got %T", msg.value)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded), "payload must be valid JSON, not base64")
	assert.Contains(t, decoded, "InfoHashes")

	var dispatchedAt pgtype.Timestamptz
	var failCount int32
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT dispatched_at, fail_count FROM torrent_retry_queue WHERE info_hash = $1 AND stage = $2`,
		"c1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", string(StageClassify)).Scan(&dispatchedAt, &failCount))
	assert.True(t, dispatchedAt.Valid, "dispatch lease must be recorded")
	assert.EqualValues(t, 1, failCount)
}

func TestIntegration_DispatchDue_LeaseExpiryCountsAsFailure(t *testing.T) {
	store, _ := newIntegrationStore(t)
	ctx := context.Background()

	cfg := testConfig()
	cfg.MaxRetries = 1
	cfg.Interval = time.Second
	cfg.DispatchLease = time.Second

	producer := &fakeProducer{}
	q := &Queue{producer: producer, storeOverride: store, logger: zap.NewNop().Sugar()}
	q.cfg.Set(cfg)

	hash := "d1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"

	require.NoError(t, q.Enqueue(ctx, StageClassify, hash, nil, assert.AnError))

	// Make the entry due, then dispatch it (this sets dispatched_at).
	_, err := store.TouchTorrentRetry(ctx, hash)
	require.NoError(t, err)

	time.Sleep(1100 * time.Millisecond)
	q.dispatchDue(ctx)

	// Lease expired with no reported outcome: the next scan counts the lost
	// attempt against the budget (fail_count 2 > max 1) and evicts.
	time.Sleep(1100 * time.Millisecond)
	q.dispatchDue(ctx)

	producer.mutex.Lock()
	defer producer.mutex.Unlock()

	var hasDelete bool
	for _, msg := range producer.messages {
		if msg.topic == kafka.TopicDeleteTorrent && msg.value == hash {
			hasDelete = true
		}
	}
	assert.True(t, hasDelete, "a lost retry outcome must eventually evict the torrent")
}

func TestIntegration_TouchTorrentRetry(t *testing.T) {
	q, conn := newIntegrationQueue(t)
	ctx := context.Background()

	hash := "e1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"
	require.NoError(t, q.Enqueue(ctx, StageEnrich, hash, []string{"id-e"}, assert.AnError))
	require.NoError(t, q.RetryNow(ctx, hash))

	var nextRetryAt time.Time
	require.NoError(t, conn.QueryRow(ctx,
		`SELECT next_retry_at FROM torrent_retry_queue WHERE info_hash = $1`, hash).Scan(&nextRetryAt))
	assert.WithinDuration(t, time.Now(), nextRetryAt, 5*time.Second)

	assert.ErrorIs(t, q.RetryNow(ctx, "0000000000000000000000000000000000000000"), pgx.ErrNoRows)
}
