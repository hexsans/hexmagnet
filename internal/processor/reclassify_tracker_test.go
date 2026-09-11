package processor

import (
	"context"
	"testing"

	"github.com/hexsans/hexmagnet/internal/jobcontrol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReclassifyProgress_InitialState(t *testing.T) {
	t.Parallel()

	p := &ReclassifyProgress{}
	total, processed, done, running, errMsg := p.Snapshot()
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, processed)
	assert.False(t, done)
	assert.False(t, running)
	assert.Empty(t, errMsg)
}

func TestReclassifyTracker_Progress_Nil(t *testing.T) {
	t.Parallel()

	tracker := NewReclassifyTracker(nil)
	total, processed, done, running, errMsg := tracker.Progress()
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, processed)
	assert.True(t, done)
	assert.False(t, running)
	assert.Empty(t, errMsg)
}

func TestReclassifyTracker_Start_AlreadyRunning(t *testing.T) {
	t.Parallel()

	tracker := NewReclassifyTracker(nil)
	tracker.mu.Lock()
	tracker.progress = &ReclassifyProgress{}
	tracker.mu.Unlock()

	err := tracker.Start(context.Background(), nil, nil, nil)
	assert.ErrorContains(t, err, "reclassify already in progress")
}

func TestReclassifyTracker_Start_MutualExclusion(t *testing.T) {
	t.Parallel()

	ctrl := jobcontrol.NewController()

	assert.True(t, ctrl.TryAcquire(jobcontrol.JobReindex))

	defer ctrl.Release(jobcontrol.JobReindex)

	tracker := NewReclassifyTracker(ctrl)

	err := tracker.Start(context.Background(), nil, nil, nil)
	require.ErrorContains(t, err, "another operation in progress")
	assert.Contains(t, err.Error(), jobcontrol.JobReindex)
}
