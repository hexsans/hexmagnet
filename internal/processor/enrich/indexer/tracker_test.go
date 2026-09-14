package indexer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/jobcontrol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestReindexTracker_Running_Nil(t *testing.T) {
	t.Parallel()

	tracker := NewReindexTracker(nil, nil)
	assert.False(t, tracker.Running())
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
