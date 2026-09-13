package processor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/jobcontrol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReclassifyTracker_Progress_Nil(t *testing.T) {
	t.Parallel()

	tracker := NewReclassifyTracker(nil, nil)
	status := tracker.Progress()
	assert.Equal(t, 0, status.Total)
	assert.Equal(t, 0, status.Processed)
	assert.True(t, status.Done)
	assert.False(t, status.Running)
	assert.False(t, status.Resumable)
	assert.Empty(t, status.Error)
}

func TestReclassifyTracker_Start_MutualExclusion(t *testing.T) {
	t.Parallel()

	ctrl := jobcontrol.NewController()

	assert.True(t, ctrl.TryAcquire(jobcontrol.JobReindex))

	defer ctrl.Release(jobcontrol.JobReindex)

	tracker := NewReclassifyTracker(ctrl, nil)

	err := tracker.Start(context.Background(), nil, nil, "", zap.NewNop().Sugar())
	require.ErrorContains(t, err, "another operation in progress")
	assert.Contains(t, err.Error(), jobcontrol.JobReindex)
}

func TestReclassifyTracker_LoadRestoresProgress(t *testing.T) {
	t.Parallel()

	barrier := time.Now().UTC().Add(-time.Hour)
	cursor := barrier.Add(time.Minute)

	store := newFakeStateStore()
	store.states[jobcontrol.KeyReclassifyState] = jobcontrol.State{
		BarrierTime:     barrier,
		CursorCreatedAt: cursor,
		CursorInfoHash:  "abc",
		Total:           100,
		Count:           25,
		Fingerprint:     "fp",
	}

	tracker := NewReclassifyTracker(nil, store)

	require.NoError(t, tracker.Load(context.Background(), "fp", nil))

	status := tracker.Progress()
	assert.Equal(t, 100, status.Total)
	assert.Equal(t, 25, status.Processed)
	assert.False(t, status.Done)
	assert.False(t, status.Running)
	assert.True(t, status.Resumable)
	assert.False(t, status.ConfigChanged)
}

func TestReclassifyTracker_LoadDetectsConfigChange(t *testing.T) {
	t.Parallel()

	store := newFakeStateStore()
	store.states[jobcontrol.KeyReclassifyState] = jobcontrol.State{
		BarrierTime: time.Now().UTC().Add(-time.Hour),
		Total:       100,
		Count:       25,
		Fingerprint: "old",
	}

	tracker := NewReclassifyTracker(nil, store)

	require.NoError(t, tracker.Load(context.Background(), "new", nil))

	status := tracker.Progress()
	assert.True(t, status.Resumable)
	assert.True(t, status.ConfigChanged)
}

func TestReclassifyTracker_LoadDoneStateIsNotResumable(t *testing.T) {
	t.Parallel()

	store := newFakeStateStore()
	store.states[jobcontrol.KeyReclassifyState] = jobcontrol.State{
		BarrierTime: time.Now().UTC().Add(-time.Hour),
		Total:       100,
		Count:       100,
		Done:        true,
		Fingerprint: "fp",
	}

	tracker := NewReclassifyTracker(nil, store)

	require.NoError(t, tracker.Load(context.Background(), "fp", nil))

	status := tracker.Progress()
	assert.True(t, status.Done)
	assert.False(t, status.Resumable)
}

func TestReclassifyTracker_Discard(t *testing.T) {
	t.Parallel()

	store := newFakeStateStore()
	store.states[jobcontrol.KeyReclassifyState] = jobcontrol.State{
		BarrierTime: time.Now().UTC().Add(-time.Hour),
		Total:       100,
		Count:       25,
		Fingerprint: "fp",
	}

	tracker := NewReclassifyTracker(nil, store)

	require.NoError(t, tracker.Load(context.Background(), "fp", nil))
	require.True(t, tracker.Progress().Resumable)

	require.NoError(t, tracker.Discard(context.Background()))

	status := tracker.Progress()
	assert.True(t, status.Done)
	assert.False(t, status.Resumable)
	assert.NotContains(t, store.states, jobcontrol.KeyReclassifyState)
}

func TestFingerprint_ChangesWithClassifierConfig(t *testing.T) {
	t.Parallel()

	base := classifier.NewDefaultConfig()

	changed := base
	changed.LLM.Prompt = "different"

	first := Fingerprint(base)
	second := Fingerprint(base)

	assert.Equal(t, first, second)
	assert.NotEqual(t, first, Fingerprint(changed))
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
