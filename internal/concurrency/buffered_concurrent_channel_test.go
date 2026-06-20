package concurrency

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBufferedConcurrentChannel_ProcessesItems(t *testing.T) {
	t.Parallel()

	ch := NewBufferedConcurrentChannel[int](10, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		mu        sync.Mutex
		processed []int
		wg        sync.WaitGroup
	)
	wg.Add(1)

	go func() {
		defer wg.Done()

		err := ch.Run(ctx, func(val int) {
			mu.Lock()

			processed = append(processed, val)
			mu.Unlock()
		})
		assert.ErrorIs(t, err, context.Canceled)
	}()

	ch.In() <- 1

	ch.In() <- 2

	ch.In() <- 3

	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	assert.ElementsMatch(t, []int{1, 2, 3}, processed)
}

func TestBufferedConcurrentChannel_RespectsConcurrency(t *testing.T) {
	t.Parallel()

	ch := NewBufferedConcurrentChannel[int](10, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		concurrent    int32
		maxConcurrent int32
		wg            sync.WaitGroup
	)
	wg.Add(1)

	go func() {
		defer wg.Done()

		_ = ch.Run(ctx, func(_ int) {
			c := atomic.AddInt32(&concurrent, 1)

			for {
				prev := atomic.LoadInt32(&maxConcurrent)
				if c <= prev {
					break
				}

				if atomic.CompareAndSwapInt32(&maxConcurrent, prev, c) {
					break
				}
			}

			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&concurrent, -1)
		})
	}()

	for i := range 6 {
		ch.In() <- i
	}

	time.Sleep(200 * time.Millisecond)
	cancel()
	wg.Wait()

	assert.LessOrEqual(t, int(maxConcurrent), 2, "should not exceed concurrency limit of 2")
}

func TestBufferedConcurrentChannel_SetConcurrency(t *testing.T) {
	t.Parallel()

	ch := NewBufferedConcurrentChannel[int](10, 1)
	ch.SetConcurrency(5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		concurrent    int32
		maxConcurrent int32
		wg            sync.WaitGroup
	)
	wg.Add(1)

	go func() {
		defer wg.Done()

		_ = ch.Run(ctx, func(_ int) {
			c := atomic.AddInt32(&concurrent, 1)

			for {
				prev := atomic.LoadInt32(&maxConcurrent)
				if c <= prev {
					break
				}

				if atomic.CompareAndSwapInt32(&maxConcurrent, prev, c) {
					break
				}
			}

			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&concurrent, -1)
		})
	}()

	for i := range 10 {
		ch.In() <- i
	}

	time.Sleep(200 * time.Millisecond)
	cancel()
	wg.Wait()

	assert.LessOrEqual(t, int(maxConcurrent), 5, "should not exceed updated concurrency of 5")
}

func TestBufferedConcurrentChannel_ContextCancel(t *testing.T) {
	t.Parallel()

	ch := NewBufferedConcurrentChannel[int](10, 2)
	ctx, cancel := context.WithCancel(context.Background())

	var processed atomic.Int32

	errCh := make(chan error, 1)

	go func() {
		errCh <- ch.Run(ctx, func(_ int) {
			processed.Add(1)
			time.Sleep(100 * time.Millisecond)
		})
	}()

	ch.In() <- 1

	time.Sleep(20 * time.Millisecond)
	cancel()

	err := <-errCh
	assert.ErrorIs(t, err, context.Canceled)
}

func TestBufferedConcurrentChannel_PanicRecovery(t *testing.T) {
	t.Parallel()

	ch := NewBufferedConcurrentChannel[int](10, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		processed atomic.Int32
		wg        sync.WaitGroup
	)
	wg.Add(1)

	go func() {
		defer wg.Done()

		_ = ch.Run(ctx, func(val int) {
			if val == 2 {
				panic("test panic")
			}

			processed.Add(1)
		})
	}()

	ch.In() <- 1

	ch.In() <- 2

	ch.In() <- 3

	time.Sleep(150 * time.Millisecond)
	cancel()
	wg.Wait()

	assert.Equal(t, int32(2), processed.Load(), "should recover from panic and continue processing")
}

func TestBufferedConcurrentChannel_InChannel(t *testing.T) {
	t.Parallel()

	ch := NewBufferedConcurrentChannel[int](5, 1)
	assert.NotNil(t, ch.In())

	ch.In() <- 42

	close(ch.In())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ch.Run(ctx, func(_ int) {})
	require.Error(t, err)
}
