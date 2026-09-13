package jobcontrol

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	sqlc "github.com/hexsans/hexmagnet/internal/database/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBatchProgress_InitialState(t *testing.T) {
	t.Parallel()

	p := &BatchProgress{}
	status := p.Snapshot()
	assert.Equal(t, 0, status.Total)
	assert.Equal(t, 0, status.Count)
	assert.False(t, status.Done)
	assert.False(t, status.Running)
	assert.False(t, status.Resumable)
	assert.Empty(t, status.Error)
}

func TestBatchProgress_SetRunning(t *testing.T) {
	t.Parallel()

	p := &BatchProgress{}
	assert.False(t, p.running)

	p.setRunning()
	assert.True(t, p.running)
	assert.True(t, p.Snapshot().Running)
}

func TestBatchProgress_SnapshotAfterUpdate(t *testing.T) {
	t.Parallel()

	p := &BatchProgress{}
	p.mu.Lock()
	p.total = 100
	p.count = 42
	p.done = true
	p.running = true
	p.errMsg = "test error"
	p.mu.Unlock()

	status := p.Snapshot()
	assert.Equal(t, 100, status.Total)
	assert.Equal(t, 42, status.Count)
	assert.True(t, status.Done)
	assert.True(t, status.Running)
	assert.Equal(t, "test error", status.Error)
}

func TestBatchProgress_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	p := &BatchProgress{}

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			p.mu.Lock()
			p.total += n
			p.count += n * 2
			p.mu.Unlock()
		}(i)
	}

	wg.Wait()

	status := p.Snapshot()
	assert.Equal(t, 45, status.Total)
	assert.Equal(t, 90, status.Count)
}

func TestBatchProgress_SnapshotThreadSafe(t *testing.T) {
	t.Parallel()

	p := &BatchProgress{}
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
				p.count++
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

func TestBatchTracker_Start_AlreadyRunning(t *testing.T) {
	t.Parallel()

	tracker := NewBatchTracker("reclassify", KeyReclassifyState, JobReclassify, "processed", nil, nil)
	tracker.mu.Lock()
	tracker.progress = &BatchProgress{running: true}
	tracker.mu.Unlock()

	err := tracker.Start(context.Background(), StartOptions{Logger: zap.NewNop().Sugar()})
	assert.ErrorContains(t, err, "reclassify already in progress")
}

func TestBatchTracker_Start_MutualExclusion(t *testing.T) {
	t.Parallel()

	ctrl := NewController()

	require.True(t, ctrl.TryAcquire(JobReindex))

	defer ctrl.Release(JobReindex)

	tracker := NewBatchTracker("reclassify", KeyReclassifyState, JobReclassify, "processed", ctrl, nil)

	err := tracker.Start(context.Background(), StartOptions{Logger: zap.NewNop().Sugar()})
	require.ErrorContains(t, err, "another operation in progress")
	assert.Contains(t, err.Error(), JobReindex)
}

func TestBatchTracker_Start_PrepareFreshFailureReleasesSlot(t *testing.T) {
	t.Parallel()

	ctrl := NewController()
	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", ctrl, nil)

	err := tracker.Start(context.Background(), StartOptions{
		PrepareFresh: func(context.Context) error {
			return errors.New("failed to recreate index: boom")
		},
		Logger: zap.NewNop().Sugar(),
	})
	require.ErrorContains(t, err, "failed to recreate index")

	assert.True(t, ctrl.TryAcquire(JobReindex), "job slot must be released after a failed fresh preparation")
	ctrl.Release(JobReindex)
}

func TestBatchTracker_Progress_Nil(t *testing.T) {
	t.Parallel()

	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", nil, nil)
	status := tracker.Progress()
	assert.Equal(t, 0, status.Total)
	assert.Equal(t, 0, status.Count)
	assert.True(t, status.Done)
	assert.False(t, status.Running)
	assert.False(t, status.Resumable)
	assert.Empty(t, status.Error)
}

func TestBatchTracker_Progress_WithProgress(t *testing.T) {
	t.Parallel()

	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", nil, nil)
	progress := &BatchProgress{}
	progress.setRunning()
	progress.mu.Lock()
	progress.total = 200
	progress.count = 50
	progress.mu.Unlock()

	tracker.mu.Lock()
	tracker.progress = progress
	tracker.mu.Unlock()

	status := tracker.Progress()
	assert.Equal(t, 200, status.Total)
	assert.Equal(t, 50, status.Count)
	assert.False(t, status.Done)
	assert.True(t, status.Running)
	assert.Empty(t, status.Error)
}

func TestBatchTracker_Running(t *testing.T) {
	t.Parallel()

	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", nil, nil)
	assert.False(t, tracker.Running())

	progress := &BatchProgress{}
	progress.setRunning()

	tracker.mu.Lock()
	tracker.progress = progress
	tracker.mu.Unlock()

	assert.True(t, tracker.Running())

	progress.mu.Lock()
	progress.done = true
	progress.mu.Unlock()

	assert.False(t, tracker.Running())
}

func TestBatchTracker_LoadRestoresProgress(t *testing.T) {
	t.Parallel()

	barrier := time.Now().UTC().Add(-time.Hour)
	cursor := barrier.Add(time.Minute)

	store := newBatchFakeStore()
	store.states[KeyReclassifyState] = State{
		BarrierTime:     barrier,
		CursorCreatedAt: cursor,
		CursorInfoHash:  "abc",
		Total:           100,
		Count:           25,
		Fingerprint:     "fp",
	}

	tracker := NewBatchTracker("reclassify", KeyReclassifyState, JobReclassify, "processed", nil, store)

	require.NoError(t, tracker.Load(context.Background(), "fp", nil))

	status := tracker.Progress()
	assert.Equal(t, 100, status.Total)
	assert.Equal(t, 25, status.Count)
	assert.False(t, status.Done)
	assert.False(t, status.Running)
	assert.True(t, status.Resumable)
	assert.False(t, status.ConfigChanged)
}

func TestBatchTracker_LoadDetectsConfigChange(t *testing.T) {
	t.Parallel()

	store := newBatchFakeStore()
	store.states[KeyReclassifyState] = State{
		BarrierTime: time.Now().UTC().Add(-time.Hour),
		Total:       100,
		Count:       25,
		Fingerprint: "old",
	}

	tracker := NewBatchTracker("reclassify", KeyReclassifyState, JobReclassify, "processed", nil, store)

	require.NoError(t, tracker.Load(context.Background(), "new", nil))

	status := tracker.Progress()
	assert.True(t, status.Resumable)
	assert.True(t, status.ConfigChanged)
}

func TestBatchTracker_LoadDoneStateIsNotResumable(t *testing.T) {
	t.Parallel()

	store := newBatchFakeStore()
	store.states[KeyReclassifyState] = State{
		BarrierTime: time.Now().UTC().Add(-time.Hour),
		Total:       100,
		Count:       100,
		Done:        true,
		Fingerprint: "fp",
	}

	tracker := NewBatchTracker("reclassify", KeyReclassifyState, JobReclassify, "processed", nil, store)

	require.NoError(t, tracker.Load(context.Background(), "fp", nil))

	status := tracker.Progress()
	assert.True(t, status.Done)
	assert.False(t, status.Resumable)
}

func TestBatchTracker_Discard(t *testing.T) {
	t.Parallel()

	store := newBatchFakeStore()
	store.states[KeyReclassifyState] = State{
		BarrierTime: time.Now().UTC().Add(-time.Hour),
		Total:       100,
		Count:       25,
		Fingerprint: "fp",
	}

	tracker := NewBatchTracker("reclassify", KeyReclassifyState, JobReclassify, "processed", nil, store)

	require.NoError(t, tracker.Load(context.Background(), "fp", nil))
	require.True(t, tracker.Progress().Resumable)

	require.NoError(t, tracker.Discard(context.Background()))

	status := tracker.Progress()
	assert.True(t, status.Done)
	assert.False(t, status.Resumable)
	assert.NotContains(t, store.states, KeyReclassifyState)
}

func TestBatchTracker_DiscardWhileRunningFails(t *testing.T) {
	t.Parallel()

	tracker := NewBatchTracker("reclassify", KeyReclassifyState, JobReclassify, "processed", nil, nil)
	tracker.mu.Lock()
	tracker.progress = &BatchProgress{running: true}
	tracker.mu.Unlock()

	err := tracker.Discard(context.Background())
	assert.ErrorContains(t, err, "reclassify in progress")
}

func TestBatchTracker_StartResumesFromStoredCursor(t *testing.T) {
	t.Parallel()

	barrier := time.Now().UTC().Add(-time.Hour)
	cursor := barrier.Add(time.Minute)

	store := newBatchFakeStore()
	store.states[KeyReclassifyState] = State{
		BarrierTime:     barrier,
		CursorCreatedAt: cursor,
		CursorInfoHash:  "cursor",
		Total:           100,
		Count:           25,
		Fingerprint:     "fp",
	}

	var (
		mu             sync.Mutex
		listParams     db.ListTorrentsPageAfterParams
		countBarrier   pgtype.Timestamptz
		listCalledWith []string
	)

	lister := &batchFakeLister{
		countFn: func(_ context.Context, b pgtype.Timestamptz) (int64, error) {
			mu.Lock()
			countBarrier = b
			mu.Unlock()

			return 100, nil
		},
		listFn: func(_ context.Context, arg db.ListTorrentsPageAfterParams) ([]db.ListTorrentsPageAfterRow, error) {
			mu.Lock()
			listParams = arg
			listCalledWith = append(listCalledWith, arg.CursorInfoHash)
			mu.Unlock()

			return []db.ListTorrentsPageAfterRow{batchRow("h1")}, nil
		},
	}

	tracker := NewBatchTracker("reclassify", KeyReclassifyState, JobReclassify, "processed", nil, store)

	require.NoError(t, tracker.Load(context.Background(), "fp", nil))
	require.True(t, tracker.Progress().Resumable)

	require.NoError(t, tracker.Start(context.Background(), StartOptions{
		Queries:     lister,
		Fingerprint: "fp",
		Run: func(_ context.Context, _ []db.ListTorrentsPageAfterRow) (int, error) {
			return 1, nil
		},
		Logger: zap.NewNop().Sugar(),
	}))

	require.Eventually(t, func() bool { return tracker.Progress().Done }, 5*time.Second, 5*time.Millisecond)

	require.Eventually(t, func() bool {
		saved, ok := store.state(KeyReclassifyState)

		return ok && saved.Done
	}, 5*time.Second, 5*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, "cursor", listParams.CursorInfoHash, "resume must continue from the stored cursor")
	assert.Equal(t, cursor, listParams.CursorCreatedAt.Time)
	assert.Equal(t, barrier, countBarrier.Time)
	assert.Equal(t, []string{"cursor"}, listCalledWith)

	saved, ok := store.state(KeyReclassifyState)
	require.True(t, ok)

	assert.True(t, saved.Done)
	assert.Equal(t, 26, saved.Count)
	assert.Equal(t, "h1", saved.CursorInfoHash)
}

func TestBatchTracker_FreshRunCompletes(t *testing.T) {
	t.Parallel()

	store := newBatchFakeStore()

	var calls int

	lister := &batchFakeLister{
		countFn: func(context.Context, pgtype.Timestamptz) (int64, error) { return 2, nil },
		listFn: func(context.Context, db.ListTorrentsPageAfterParams) ([]db.ListTorrentsPageAfterRow, error) {
			calls++
			if calls > 1 {
				return nil, nil
			}

			return []db.ListTorrentsPageAfterRow{batchRow("h1"), batchRow("h2")}, nil
		},
	}

	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", nil, store)

	require.NoError(t, tracker.Start(context.Background(), StartOptions{
		Queries:     lister,
		Fingerprint: "fp",
		Run: func(_ context.Context, rows []db.ListTorrentsPageAfterRow) (int, error) {
			return len(rows), nil
		},
		Logger: zap.NewNop().Sugar(),
	}))

	require.Eventually(t, func() bool { return tracker.Progress().Done }, 5*time.Second, 5*time.Millisecond)

	require.Eventually(t, func() bool {
		saved, ok := store.state(KeyReindexState)

		return ok && saved.Done
	}, 5*time.Second, 5*time.Millisecond)

	status := tracker.Progress()
	assert.Equal(t, 2, status.Total)
	assert.Equal(t, 2, status.Count)
	assert.False(t, status.Resumable)

	saved, ok := store.state(KeyReindexState)
	require.True(t, ok)

	assert.True(t, saved.Done)
	assert.Equal(t, 2, saved.Count)
	assert.False(t, saved.BarrierTime.IsZero())
}

func TestBatchTracker_RunErrorFailsProgress(t *testing.T) {
	t.Parallel()

	store := newBatchFakeStore()

	lister := &batchFakeLister{
		countFn: func(context.Context, pgtype.Timestamptz) (int64, error) { return 1, nil },
		listFn: func(context.Context, db.ListTorrentsPageAfterParams) ([]db.ListTorrentsPageAfterRow, error) {
			return []db.ListTorrentsPageAfterRow{batchRow("h1")}, nil
		},
	}

	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", nil, store)

	require.NoError(t, tracker.Start(context.Background(), StartOptions{
		Queries:     lister,
		Fingerprint: "fp",
		Run: func(context.Context, []db.ListTorrentsPageAfterRow) (int, error) {
			return 0, errors.New("failed to build bulk body: boom")
		},
		Logger: zap.NewNop().Sugar(),
	}))

	require.Eventually(t, func() bool {
		return tracker.Progress().Error != ""
	}, 5*time.Second, 5*time.Millisecond)

	status := tracker.Progress()
	assert.False(t, status.Done)
	assert.False(t, status.Running)
	assert.Contains(t, status.Error, "failed to build bulk body")
}

func TestBatchTracker_PanicRecovery(t *testing.T) {
	t.Parallel()

	store := newBatchFakeStore()

	lister := &batchFakeLister{
		countFn: func(context.Context, pgtype.Timestamptz) (int64, error) {
			panic("unexpected boom")
		},
		listFn: func(context.Context, db.ListTorrentsPageAfterParams) ([]db.ListTorrentsPageAfterRow, error) {
			return nil, nil
		},
	}

	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", nil, store)

	require.NoError(t, tracker.Start(context.Background(), StartOptions{
		Queries:     lister,
		Fingerprint: "fp",
		Run: func(context.Context, []db.ListTorrentsPageAfterRow) (int, error) {
			return 0, nil
		},
		Logger: zap.NewNop().Sugar(),
	}))

	require.Eventually(t, func() bool {
		return tracker.Progress().Error != ""
	}, 5*time.Second, 5*time.Millisecond)

	status := tracker.Progress()
	assert.False(t, status.Done)
	assert.False(t, status.Running)
	assert.Contains(t, status.Error, "reindex panic")
	assert.Contains(t, status.Error, "unexpected boom")
}

func TestBatchTracker_ContextCanceledBeforeCounting(t *testing.T) {
	t.Parallel()

	store := newBatchFakeStore()

	lister := &batchFakeLister{
		countFn: func(ctx context.Context, _ pgtype.Timestamptz) (int64, error) {
			return 0, ctx.Err()
		},
		listFn: func(context.Context, db.ListTorrentsPageAfterParams) ([]db.ListTorrentsPageAfterRow, error) {
			return nil, nil
		},
	}

	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", nil, store)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, tracker.Start(ctx, StartOptions{
		Queries:     lister,
		Fingerprint: "fp",
		Run: func(context.Context, []db.ListTorrentsPageAfterRow) (int, error) {
			return 0, nil
		},
		Logger: zap.NewNop().Sugar(),
	}))

	require.Eventually(t, func() bool {
		return tracker.Progress().Error != ""
	}, 5*time.Second, 5*time.Millisecond)

	status := tracker.Progress()
	assert.False(t, status.Done)
	assert.False(t, status.Running)
	assert.Contains(t, status.Error, "failed to count torrents")
	assert.Contains(t, status.Error, "context canceled")
}

func TestBatchTracker_ContextCanceledDuringLoop(t *testing.T) {
	t.Parallel()

	store := newBatchFakeStore()

	ctx, cancel := context.WithCancel(context.Background())

	lister := &batchFakeLister{
		countFn: func(context.Context, pgtype.Timestamptz) (int64, error) {
			cancel()

			return 100, nil
		},
		listFn: func(context.Context, db.ListTorrentsPageAfterParams) ([]db.ListTorrentsPageAfterRow, error) {
			return nil, nil
		},
	}

	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", nil, store)

	require.NoError(t, tracker.Start(ctx, StartOptions{
		Queries:     lister,
		Fingerprint: "fp",
		Run: func(context.Context, []db.ListTorrentsPageAfterRow) (int, error) {
			return 0, nil
		},
		Logger: zap.NewNop().Sugar(),
	}))

	require.Eventually(t, func() bool {
		return tracker.Progress().Error != ""
	}, 5*time.Second, 5*time.Millisecond)

	status := tracker.Progress()
	assert.Equal(t, 100, status.Total)
	assert.Equal(t, 0, status.Count)
	assert.False(t, status.Done)
	assert.Contains(t, status.Error, "reindex canceled")
}

func TestBatchTracker_RunErrorPausesAndContinueResumes(t *testing.T) {
	t.Parallel()

	store := newBatchFakeStore()

	lister := &batchFakeLister{
		countFn: func(context.Context, pgtype.Timestamptz) (int64, error) { return 1, nil },
		listFn: func(context.Context, db.ListTorrentsPageAfterParams) ([]db.ListTorrentsPageAfterRow, error) {
			return []db.ListTorrentsPageAfterRow{batchRow("h1")}, nil
		},
	}

	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", nil, store)

	var attempts atomic.Int32

	run := func(context.Context, []db.ListTorrentsPageAfterRow) (int, error) {
		if attempts.Add(1) == 1 {
			return 0, errors.New("embed batch: boom")
		}

		return 1, nil
	}

	startOpts := func() StartOptions {
		return StartOptions{
			Queries:     lister,
			Fingerprint: "fp",
			Run:         run,
			Logger:      zap.NewNop().Sugar(),
		}
	}

	require.NoError(t, tracker.Start(context.Background(), startOpts()))

	require.Eventually(t, func() bool {
		status := tracker.Progress()

		return status.Error != "" && !status.Running
	}, 5*time.Second, 5*time.Millisecond)

	paused := tracker.Progress()
	assert.False(t, paused.Done)
	assert.True(t, paused.Resumable, "a failed run must stay resumable so the operator can continue")
	assert.Contains(t, paused.Error, "embed batch")

	require.NoError(t, tracker.Start(context.Background(), startOpts()))

	require.Eventually(t, func() bool {
		return tracker.Progress().Done
	}, 5*time.Second, 5*time.Millisecond)

	done := tracker.Progress()
	assert.Equal(t, 1, done.Count)
	assert.False(t, done.Resumable)
	assert.Equal(t, int32(2), attempts.Load(), "continue must retry the same page")
}

func TestBatchTracker_OnFinishCalled(t *testing.T) {
	t.Parallel()

	store := newBatchFakeStore()

	lister := &batchFakeLister{
		countFn: func(context.Context, pgtype.Timestamptz) (int64, error) { return 0, nil },
		listFn: func(context.Context, db.ListTorrentsPageAfterParams) ([]db.ListTorrentsPageAfterRow, error) {
			return nil, nil
		},
	}

	tracker := NewBatchTracker("reindex", KeyReindexState, JobReindex, "indexed", nil, store)
	finished := make(chan struct{})

	require.NoError(t, tracker.Start(context.Background(), StartOptions{
		Queries:     lister,
		Fingerprint: "fp",
		Run: func(context.Context, []db.ListTorrentsPageAfterRow) (int, error) {
			return 0, nil
		},
		OnFinish: func() {
			close(finished)
		},
		Logger: zap.NewNop().Sugar(),
	}))

	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("OnFinish was not called")
	}
}

func batchRow(infoHash string) db.ListTorrentsPageAfterRow {
	return db.ListTorrentsPageAfterRow{
		Torrent: sqlc.Torrent{
			InfoHash:  infoHash,
			CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		},
	}
}

type batchFakeLister struct {
	countFn func(ctx context.Context, barrierTime pgtype.Timestamptz) (int64, error)
	listFn  func(ctx context.Context, arg db.ListTorrentsPageAfterParams) ([]db.ListTorrentsPageAfterRow, error)
}

func (f *batchFakeLister) CountTorrentsBefore(ctx context.Context, barrierTime pgtype.Timestamptz) (int64, error) {
	return f.countFn(ctx, barrierTime)
}

func (f *batchFakeLister) ListTorrentsPageAfter(
	ctx context.Context,
	arg db.ListTorrentsPageAfterParams,
) ([]db.ListTorrentsPageAfterRow, error) {
	return f.listFn(ctx, arg)
}

type batchFakeStore struct {
	mu     sync.Mutex
	states map[string]State
}

func newBatchFakeStore() *batchFakeStore {
	return &batchFakeStore{states: make(map[string]State)}
}

func (f *batchFakeStore) Load(_ context.Context, key string) (State, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	state, ok := f.states[key]

	return state, ok, nil
}

func (f *batchFakeStore) Save(_ context.Context, key string, state State) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.states[key] = state

	return nil
}

func (f *batchFakeStore) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.states, key)

	return nil
}

// state returns a persisted state with the store lock held, so tests can read
// it while the tracker goroutine is still saving.
func (f *batchFakeStore) state(key string) (State, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	state, ok := f.states[key]

	return state, ok
}
