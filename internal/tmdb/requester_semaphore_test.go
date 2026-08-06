package tmdb

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

func TestRequesterSemaphore_AcquireAndRelease(t *testing.T) {
	t.Parallel()

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			return &resty.Response{}, nil
		},
	}
	r := requesterSemaphore{
		requester: inner,
		semaphore: semaphore.NewWeighted(2),
	}
	_, err := r.Request(context.Background(), "/test", nil, nil)
	assert.NoError(t, err)
}

func TestRequesterSemaphore_ContextCancelled(t *testing.T) {
	t.Parallel()

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			return &resty.Response{}, nil
		},
	}
	r := requesterSemaphore{
		requester: inner,
		semaphore: semaphore.NewWeighted(0),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := r.Request(ctx, "/test", nil, nil)
	assert.Error(t, err)
}

func TestRequesterSemaphore_ConcurrentRequests_LimitedBySemaphore(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex

	concurrent := 0
	maxConcurrent := 0

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			mu.Lock()

			concurrent++
			if concurrent > maxConcurrent {
				maxConcurrent = concurrent
			}
			mu.Unlock()

			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			concurrent--
			mu.Unlock()

			return &resty.Response{}, nil
		},
	}

	r := requesterSemaphore{
		requester: inner,
		semaphore: semaphore.NewWeighted(2),
	}

	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := r.Request(context.Background(), "/test", nil, nil)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	assert.LessOrEqual(t, maxConcurrent, 2)
}

func TestRequesterSemaphore_SingleConcurrent(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex

	concurrent := 0
	maxConcurrent := 0

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			mu.Lock()

			concurrent++
			if concurrent > maxConcurrent {
				maxConcurrent = concurrent
			}
			mu.Unlock()

			time.Sleep(30 * time.Millisecond)

			mu.Lock()
			concurrent--
			mu.Unlock()

			return &resty.Response{}, nil
		},
	}

	r := requesterSemaphore{
		requester: inner,
		semaphore: semaphore.NewWeighted(1),
	}

	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := r.Request(context.Background(), "/test", nil, nil)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	assert.LessOrEqual(t, maxConcurrent, 1)
}

func TestRequesterSemaphore_UnlimitedConcurrent(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex

	concurrent := 0
	maxConcurrent := 0

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			mu.Lock()

			concurrent++
			if concurrent > maxConcurrent {
				maxConcurrent = concurrent
			}
			mu.Unlock()

			time.Sleep(20 * time.Millisecond)

			mu.Lock()
			concurrent--
			mu.Unlock()

			return &resty.Response{}, nil
		},
	}

	r := requesterSemaphore{
		requester: inner,
		semaphore: semaphore.NewWeighted(10),
	}

	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := r.Request(context.Background(), "/test", nil, nil)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	assert.Greater(t, maxConcurrent, 1)
}
