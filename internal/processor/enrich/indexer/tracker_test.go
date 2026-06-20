package indexer

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/hexsans/hexmagnet/internal/database/db"
	sqlc "github.com/hexsans/hexmagnet/internal/database/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestReindexProgress_InitialState(t *testing.T) {
	t.Parallel()

	p := &ReindexProgress{}
	total, indexed, done, running, errMsg := p.Snapshot()
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, indexed)
	assert.False(t, done)
	assert.False(t, running)
	assert.Empty(t, errMsg)
}

func TestReindexProgress_SetRunning(t *testing.T) {
	t.Parallel()

	p := &ReindexProgress{}
	assert.False(t, p.running)
	p.setRunning()
	assert.True(t, p.running)

	_, _, _, running, _ := p.Snapshot()
	assert.True(t, running)
}

func TestReindexProgress_SnapshotAfterUpdate(t *testing.T) {
	t.Parallel()

	p := &ReindexProgress{}
	p.mu.Lock()
	p.total = 100
	p.indexed = 42
	p.done = true
	p.running = true
	p.errMsg = "test error"
	p.mu.Unlock()

	total, indexed, done, running, errMsg := p.Snapshot()
	assert.Equal(t, 100, total)
	assert.Equal(t, 42, indexed)
	assert.True(t, done)
	assert.True(t, running)
	assert.Equal(t, "test error", errMsg)
}

func TestReindexProgress_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	p := &ReindexProgress{}

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			p.mu.Lock()
			p.total += n
			p.indexed += n * 2
			p.mu.Unlock()
		}(i)
	}

	wg.Wait()

	total, indexed, _, _, _ := p.Snapshot()
	assert.Equal(t, 45, total)
	assert.Equal(t, 90, indexed)
}

func TestReindexProgress_SnapshotThreadSafe(t *testing.T) {
	t.Parallel()

	p := &ReindexProgress{}
	started := make(chan struct{})
	done := make(chan struct{})

	go func() {
		close(started)

		for {
			select {
			case <-done:
				return
			default:
				p.mu.Lock()
				p.indexed++
				p.mu.Unlock()
			}
		}
	}()

	<-started

	for range 1000 {
		_, _, _, _, _ = p.Snapshot()
	}

	close(done)

	_, _, _, _, _ = p.Snapshot()
}

func TestReindexTracker_Start_AlreadyRunning(t *testing.T) {
	t.Parallel()

	tracker := NewReindexTracker()
	tracker.mu.Lock()
	tracker.progress = &ReindexProgress{}
	tracker.mu.Unlock()

	err := tracker.Start(context.Background(), nil, nil, nil, 0, nil)
	assert.ErrorContains(t, err, "reindex already in progress")
}

func TestReindexTracker_Progress_Nil(t *testing.T) {
	t.Parallel()

	tracker := NewReindexTracker()
	total, indexed, done, running, errMsg := tracker.Progress()
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, indexed)
	assert.True(t, done)
	assert.False(t, running)
	assert.Empty(t, errMsg)
}

func TestReindexTracker_Progress_WithProgress(t *testing.T) {
	t.Parallel()

	tracker := NewReindexTracker()
	progress := &ReindexProgress{}
	progress.setRunning()
	progress.mu.Lock()
	progress.total = 200
	progress.indexed = 50
	progress.mu.Unlock()

	tracker.mu.Lock()
	tracker.progress = progress
	tracker.mu.Unlock()

	total, indexed, done, running, errMsg := tracker.Progress()
	assert.Equal(t, 200, total)
	assert.Equal(t, 50, indexed)
	assert.False(t, done)
	assert.True(t, running)
	assert.Empty(t, errMsg)
}

func TestReindexProgress_RunningReflectsActiveState(t *testing.T) {
	t.Parallel()

	p := &ReindexProgress{}
	_, _, done, running, _ := p.Snapshot()
	assert.False(t, done)
	assert.False(t, running)

	p.setRunning()
	_, _, done, running, _ = p.Snapshot()
	assert.False(t, done)
	assert.True(t, running)

	p.mu.Lock()
	p.done = true
	p.mu.Unlock()
	_, _, done, running, _ = p.Snapshot()
	assert.True(t, done)
	assert.True(t, running)
}

func TestRunReindex_PanicRecovery(t *testing.T) {
	t.Parallel()

	progress := &ReindexProgress{}
	ctx := context.Background()

	mockDB := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{
				scanFn: func(_ ...any) error {
					panic("unexpected boom")
				},
			}
		},
	}

	sqlcQ := sqlc.New(mockDB)
	q := &db.Queries{Queries: sqlcQ}
	logger := zap.NewNop().Sugar()

	runReindex(ctx, q, nil, nil, progress, logger)

	_, _, done, running, errMsg := progress.Snapshot()
	assert.True(t, done)
	assert.True(t, running)
	assert.Contains(t, errMsg, "reindex panic")
	assert.Contains(t, errMsg, "unexpected boom")
}

func TestConfigNotifier_SubscribeNotify(t *testing.T) {
	t.Parallel()

	n := NewConfigNotifier()

	ch, unsubscribe := n.Subscribe()
	defer unsubscribe()

	n.Notify()

	select {
	case <-ch:
	default:
		t.Fatal("expected notification")
	}
}

func TestConfigNotifier_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	n := NewConfigNotifier()

	ch1, unsub1 := n.Subscribe()
	defer unsub1()

	ch2, unsub2 := n.Subscribe()
	defer unsub2()

	n.Notify()

	select {
	case <-ch1:
	default:
		t.Fatal("subscriber 1 missed notification")
	}

	select {
	case <-ch2:
	default:
		t.Fatal("subscriber 2 missed notification")
	}
}

func TestConfigNotifier_BufferedChannel(t *testing.T) {
	t.Parallel()

	n := NewConfigNotifier()

	ch, unsubscribe := n.Subscribe()
	defer unsubscribe()

	n.Notify()
	n.Notify()

	select {
	case <-ch:
	default:
		t.Fatal("expected at least one notification")
	}
}

func TestConfigNotifier_NoSubscribers(t *testing.T) {
	t.Parallel()

	n := NewConfigNotifier()
	n.Notify()
}

func TestConfigNotifier_Unsubscribed(t *testing.T) {
	t.Parallel()

	n := NewConfigNotifier()

	ch, unsubscribe := n.Subscribe()
	unsubscribe()

	n.Notify()

	select {
	case v := <-ch:
		if v != struct{}{} {
			t.Fatal("unexpected value from closed channel")
		}
	default:
	}
}

type mockRow struct {
	scanFn func(dest ...any) error
}

func (m *mockRow) Scan(dest ...any) error {
	return m.scanFn(dest...)
}

type mockDBTX struct {
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (*mockDBTX) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (*mockDBTX) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query call")
}

func (m *mockDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return m.queryRowFn(ctx, sql, args...)
}

func TestRunReindex_ContextCanceledBeforeCounting(t *testing.T) {
	t.Parallel()

	progress := &ReindexProgress{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockDB := &mockDBTX{
		queryRowFn: func(ctx context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{
				scanFn: func(_ ...any) error {
					return ctx.Err()
				},
			}
		},
	}

	sqlcQ := sqlc.New(mockDB)
	q := &db.Queries{Queries: sqlcQ}
	logger := zap.NewNop().Sugar()

	runReindex(ctx, q, nil, nil, progress, logger)

	_, _, done, running, errMsg := progress.Snapshot()
	assert.Equal(t, 0, progress.total)
	assert.True(t, done)
	assert.True(t, running)
	assert.Contains(t, errMsg, "failed to count torrents")
	assert.Contains(t, errMsg, "context canceled")
}

func TestRunReindex_ContextCanceledDuringLoop(t *testing.T) {
	t.Parallel()

	progress := &ReindexProgress{}
	ctx, cancel := context.WithCancel(context.Background())

	mockDB := &mockDBTX{
		queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{
				scanFn: func(dest ...any) error {
					*dest[0].(*int64) = 100

					cancel()

					return nil
				},
			}
		},
	}

	sqlcQ := sqlc.New(mockDB)
	q := &db.Queries{Queries: sqlcQ}
	logger := zap.NewNop().Sugar()

	runReindex(ctx, q, nil, nil, progress, logger)

	_, _, done, running, errMsg := progress.Snapshot()
	assert.Equal(t, 100, progress.total)
	assert.Equal(t, 0, progress.indexed)
	assert.True(t, done)
	assert.True(t, running)
	assert.Contains(t, errMsg, "reindex canceled")
}
