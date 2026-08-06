package queuemetrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBucket_ZeroValues(t *testing.T) {
	t.Parallel()

	var b Bucket
	assert.Empty(t, b.Queue)
	assert.Empty(t, b.Status)
	assert.True(t, b.CreatedAtBucket.IsZero())
	assert.Nil(t, b.RanAtBucket)
	assert.Equal(t, uint(0), b.Count)
	assert.Nil(t, b.Latency)
}

func TestBucket_FullValues(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ranAt := now.Add(time.Hour)
	latency := 5 * time.Second

	b := Bucket{
		Queue:           "test_queue",
		Status:          "completed",
		CreatedAtBucket: now,
		RanAtBucket:     &ranAt,
		Count:           42,
		Latency:         &latency,
	}

	assert.Equal(t, "test_queue", b.Queue)
	assert.Equal(t, "completed", b.Status)
	assert.Equal(t, now, b.CreatedAtBucket)
	assert.Equal(t, &ranAt, b.RanAtBucket)
	assert.Equal(t, uint(42), b.Count)
	assert.Equal(t, &latency, b.Latency)
}

func TestBucket_NilPointers(t *testing.T) {
	t.Parallel()

	b := Bucket{
		Queue:  "queue",
		Status: "pending",
		Count:  1,
	}
	assert.Nil(t, b.RanAtBucket)
	assert.Nil(t, b.Latency)
}
