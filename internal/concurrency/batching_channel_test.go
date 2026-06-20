package concurrency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readWithTimeout[T any](ch BatchingChannel[T], timeout time.Duration) ([]T, bool) {
	tm := time.NewTimer(timeout)
	defer tm.Stop()

	select {
	case batch := <-ch.Out():
		return batch, true
	case <-tm.C:
		return nil, false
	}
}

func TestBatchingChannel_FlushByBatchSize(t *testing.T) {
	t.Parallel()

	ch := NewBatchingChannel[int](10, 3, 50*time.Millisecond)

	for i := range 6 {
		ch.In() <- i
	}

	batch1, ok := readWithTimeout(ch, 200*time.Millisecond)
	require.True(t, ok)
	assert.Equal(t, []int{0, 1, 2}, batch1)

	batch2, ok := readWithTimeout(ch, 200*time.Millisecond)
	require.True(t, ok)
	assert.Equal(t, []int{3, 4, 5}, batch2)
}

func TestBatchingChannel_FlushByTime(t *testing.T) {
	t.Parallel()

	ch := NewBatchingChannel[int](10, 100, 50*time.Millisecond)

	ch.In() <- 1

	ch.In() <- 2

	batch1, ok := readWithTimeout(ch, 150*time.Millisecond)
	require.True(t, ok)
	assert.Equal(t, []int{1, 2}, batch1)

	ch.In() <- 3

	batch2, ok := readWithTimeout(ch, 150*time.Millisecond)
	require.True(t, ok)
	assert.Equal(t, []int{3}, batch2)
}

func TestBatchingChannel_EmptyChannel(t *testing.T) {
	t.Parallel()

	ch := NewBatchingChannel[int](10, 5, 10*time.Millisecond)

	_, ok := readWithTimeout(ch, 50*time.Millisecond)
	assert.False(t, ok)
}

func TestBatchingChannel_SingleItem(t *testing.T) {
	t.Parallel()

	ch := NewBatchingChannel[int](10, 5, 50*time.Millisecond)

	ch.In() <- 99

	batch, ok := readWithTimeout(ch, 150*time.Millisecond)
	require.True(t, ok)
	assert.Equal(t, []int{99}, batch)
}

func TestBatchingChannel_StringType(t *testing.T) {
	t.Parallel()

	ch := NewBatchingChannel[string](10, 2, 100*time.Millisecond)

	ch.In() <- "a"

	ch.In() <- "b"

	batch1, ok := readWithTimeout(ch, 200*time.Millisecond)
	require.True(t, ok)
	assert.Equal(t, []string{"a", "b"}, batch1)

	ch.In() <- "c"

	ch.In() <- "d"

	batch2, ok := readWithTimeout(ch, 200*time.Millisecond)
	require.True(t, ok)
	assert.Equal(t, []string{"c", "d"}, batch2)
}

func TestBatchingChannel_FlushResetsTicker(t *testing.T) {
	t.Parallel()

	ch := NewBatchingChannel[int](10, 3, time.Hour)

	ch.In() <- 1

	ch.In() <- 2

	ch.In() <- 3

	batch1, ok := readWithTimeout(ch, 100*time.Millisecond)
	require.True(t, ok)
	assert.Equal(t, []int{1, 2, 3}, batch1)

	ch.In() <- 4

	ch.In() <- 5

	ch.In() <- 6

	batch2, ok := readWithTimeout(ch, 100*time.Millisecond)
	require.True(t, ok)
	assert.Equal(t, []int{4, 5, 6}, batch2)
}

func TestBatchingChannel_LargeCapacity(t *testing.T) {
	t.Parallel()

	ch := NewBatchingChannel[int](100, 10, 50*time.Millisecond)

	for i := range 25 {
		ch.In() <- i
	}

	var total int

	for range 2 {
		batch, ok := readWithTimeout(ch, 200*time.Millisecond)
		require.True(t, ok)

		total += len(batch)
	}

	batch3, ok := readWithTimeout(ch, 200*time.Millisecond)
	if ok {
		total += len(batch3)
	}

	assert.Equal(t, 25, total)
}
