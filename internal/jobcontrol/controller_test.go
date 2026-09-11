package jobcontrol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestController_ExclusiveAcquire(t *testing.T) {
	t.Parallel()

	c := NewController()

	assert.False(t, c.Paused())
	assert.Empty(t, c.Reason())

	assert.True(t, c.TryAcquire(JobReindex))
	assert.True(t, c.Paused())
	assert.Equal(t, JobReindex, c.Reason())

	assert.False(t, c.TryAcquire(JobReclassify), "second job must be rejected while one is active")
	assert.Equal(t, JobReindex, c.Reason())

	c.Release(JobReindex)
	assert.False(t, c.Paused())
	assert.Empty(t, c.Reason())

	assert.True(t, c.TryAcquire(JobReclassify))
	assert.Equal(t, JobReclassify, c.Reason())
}

func TestController_ReleaseUnknownIsSafe(t *testing.T) {
	t.Parallel()

	c := NewController()
	c.Release("does-not-exist")
	assert.False(t, c.Paused())
}
