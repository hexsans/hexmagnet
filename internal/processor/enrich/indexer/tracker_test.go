package indexer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	sqlc "github.com/hexsans/hexmagnet/internal/database/db/sqlc"
	"github.com/hexsans/hexmagnet/internal/jobcontrol"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReindexProgress_InitialState(t *testing.T) {
	t.Parallel()

	p := &ReindexProgress{}
	status := p.Snapshot()
	assert.Equal(t, 0, status.Total)
	assert.Equal(t, 0, status.Indexed)
	assert.False(t, status.Done)
	assert.False(t, status.Running)
	assert.False(t, status.Resumable)
	assert.Empty(t, status.Error)
}

func TestReindexProgress_SetRunning(t *testing.T) {
	t.Parallel()

	p := &ReindexProgress{}
	assert.False(t, p.running)
	p.setRunning()
	assert.True(t, p.running)

	assert.True(t, p.Snapshot().Running)
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

	status := p.Snapshot()
	assert.Equal(t, 100, status.Total)
	assert.Equal(t, 42, status.Indexed)
	assert.True(t, status.Done)
	assert.True(t, status.Running)
	assert.Equal(t, "test error", status.Error)
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

	status := p.Snapshot()
	assert.Equal(t, 45, status.Total)
	assert.Equal(t, 90, status.Indexed)
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
		_ = p.Snapshot()
	}

	close(done)

	_ = p.Snapshot()
}

func TestReindexTracker_Start_AlreadyRunning(t *testing.T) {
	t.Parallel()

	tracker := NewReindexTracker(nil, nil)
	tracker.mu.Lock()
	tracker.progress = &ReindexProgress{running: true}
	tracker.mu.Unlock()

	err := tracker.Start(context.Background(), nil, nil, nil, 0, 0, "", false, nil)
	assert.ErrorContains(t, err, "reindex already in progress")
}

func TestReindexTracker_Progress_Nil(t *testing.T) {
	t.Parallel()

	tracker := NewReindexTracker(nil, nil)
	status := tracker.Progress()
	assert.Equal(t, 0, status.Total)
	assert.Equal(t, 0, status.Indexed)
	assert.True(t, status.Done)
	assert.False(t, status.Running)
	assert.False(t, status.Resumable)
	assert.Empty(t, status.Error)
}

func TestReindexTracker_Progress_WithProgress(t *testing.T) {
	t.Parallel()

	tracker := NewReindexTracker(nil, nil)
	progress := &ReindexProgress{}
	progress.setRunning()
	progress.mu.Lock()
	progress.total = 200
	progress.indexed = 50
	progress.mu.Unlock()

	tracker.mu.Lock()
	tracker.progress = progress
	tracker.mu.Unlock()

	status := tracker.Progress()
	assert.Equal(t, 200, status.Total)
	assert.Equal(t, 50, status.Indexed)
	assert.False(t, status.Done)
	assert.True(t, status.Running)
	assert.Empty(t, status.Error)
}

func TestReindexTracker_Running_Nil(t *testing.T) {
	t.Parallel()

	tracker := NewReindexTracker(nil, nil)
	assert.False(t, tracker.Running())
}

func TestReindexTracker_Running_Active(t *testing.T) {
	t.Parallel()

	tracker := NewReindexTracker(nil, nil)
	progress := &ReindexProgress{}
	progress.setRunning()

	tracker.mu.Lock()
	tracker.progress = progress
	tracker.mu.Unlock()

	assert.True(t, tracker.Running())
}

func TestReindexTracker_Running_Done(t *testing.T) {
	t.Parallel()

	tracker := NewReindexTracker(nil, nil)
	progress := &ReindexProgress{}
	progress.setRunning()
	progress.mu.Lock()
	progress.done = true
	progress.mu.Unlock()

	tracker.mu.Lock()
	tracker.progress = progress
	tracker.mu.Unlock()

	assert.False(t, tracker.Running())
}

func TestReindexProgress_RunningReflectsActiveState(t *testing.T) {
	t.Parallel()

	p := &ReindexProgress{}
	status := p.Snapshot()
	assert.False(t, status.Done)
	assert.False(t, status.Running)

	p.setRunning()
	status = p.Snapshot()
	assert.False(t, status.Done)
	assert.True(t, status.Running)

	p.mu.Lock()
	p.done = true
	p.mu.Unlock()
	status = p.Snapshot()
	assert.True(t, status.Done)
	assert.True(t, status.Running)
}

func TestReindexTracker_LoadRestoresProgress(t *testing.T) {
	t.Parallel()

	barrier := time.Now().UTC().Add(-time.Hour)
	cursor := barrier.Add(time.Minute)

	store := newFakeStateStore()
	store.states[jobcontrol.KeyReindexState] = jobcontrol.State{
		BarrierTime:     barrier,
		CursorCreatedAt: cursor,
		CursorInfoHash:  "abc",
		Total:           100,
		Count:           25,
		Fingerprint:     "fp",
	}

	tracker := NewReindexTracker(nil, store)

	require.NoError(t, tracker.Load(context.Background(), "fp", nil))

	status := tracker.Progress()
	assert.Equal(t, 100, status.Total)
	assert.Equal(t, 25, status.Indexed)
	assert.False(t, status.Done)
	assert.True(t, status.Resumable)
	assert.False(t, status.ConfigChanged)
}

func TestReindexTracker_LoadDetectsConfigChange(t *testing.T) {
	t.Parallel()

	store := newFakeStateStore()
	store.states[jobcontrol.KeyReindexState] = jobcontrol.State{
		BarrierTime: time.Now().UTC().Add(-time.Hour),
		Total:       100,
		Count:       25,
		Fingerprint: "old",
	}

	tracker := NewReindexTracker(nil, store)

	require.NoError(t, tracker.Load(context.Background(), "new", nil))

	status := tracker.Progress()
	assert.True(t, status.Resumable)
	assert.True(t, status.ConfigChanged)
}

func TestReindexTracker_Discard(t *testing.T) {
	t.Parallel()

	store := newFakeStateStore()
	store.states[jobcontrol.KeyReindexState] = jobcontrol.State{
		BarrierTime: time.Now().UTC().Add(-time.Hour),
		Total:       100,
		Count:       25,
		Fingerprint: "fp",
	}

	tracker := NewReindexTracker(nil, store)

	require.NoError(t, tracker.Load(context.Background(), "fp", nil))
	require.True(t, tracker.Progress().Resumable)

	require.NoError(t, tracker.Discard(context.Background()))

	status := tracker.Progress()
	assert.True(t, status.Done)
	assert.False(t, status.Resumable)
	assert.NotContains(t, store.states, jobcontrol.KeyReindexState)
}

func TestFingerprint_ChangesWithSearchConfig(t *testing.T) {
	t.Parallel()

	base := DefaultSearchConfig()

	changed := base
	changed.Elasticsearch.Embedding.Dimensions = 2048

	first := Fingerprint(base)
	second := Fingerprint(base)

	assert.Equal(t, first, second)
	assert.NotEqual(t, first, Fingerprint(changed))
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

	runReindex(ctx, q, nil, nil, 0, progress, nil, func() {}, logger)

	status := progress.Snapshot()
	assert.False(t, status.Done)
	assert.False(t, status.Running)
	assert.Contains(t, status.Error, "reindex panic")
	assert.Contains(t, status.Error, "unexpected boom")
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

	runReindex(ctx, q, nil, nil, 0, progress, nil, func() {}, logger)

	status := progress.Snapshot()
	assert.Equal(t, 0, progress.total)
	assert.False(t, status.Done)
	assert.False(t, status.Running)
	assert.Contains(t, status.Error, "failed to count torrents")
	assert.Contains(t, status.Error, "context canceled")
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

	runReindex(ctx, q, nil, nil, 0, progress, nil, func() {}, logger)

	status := progress.Snapshot()
	assert.Equal(t, 100, progress.total)
	assert.Equal(t, 0, progress.indexed)
	assert.False(t, status.Done)
	assert.False(t, status.Running)
	assert.Contains(t, status.Error, "reindex canceled")
}

type fakeStateStore struct {
	mu     sync.Mutex
	states map[string]jobcontrol.State
}

func newFakeStateStore() *fakeStateStore {
	return &fakeStateStore{states: make(map[string]jobcontrol.State)}
}

func (f *fakeStateStore) Load(_ context.Context, key string) (jobcontrol.State, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	state, ok := f.states[key]

	return state, ok, nil
}

func (f *fakeStateStore) Save(_ context.Context, key string, state jobcontrol.State) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.states[key] = state

	return nil
}

func (f *fakeStateStore) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.states, key)

	return nil
}
