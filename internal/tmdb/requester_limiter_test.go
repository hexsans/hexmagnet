package tmdb

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestRequesterLimiter_PassesRequest(t *testing.T) {
	t.Parallel()

	called := false
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			called = true
			return &resty.Response{}, nil
		},
	}
	r := requesterLimiter{
		requester: inner,
		limiter:   rate.NewLimiter(rate.Inf, 0),
	}
	_, err := r.Request(context.Background(), "/test", nil, nil)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRequesterLimiter_MultipleRequestsWithInfiniteRate(t *testing.T) {
	t.Parallel()

	callCount := 0

	var mu sync.Mutex

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			mu.Lock()
			callCount++
			mu.Unlock()

			return &resty.Response{}, nil
		},
	}

	r := requesterLimiter{
		requester: inner,
		limiter:   rate.NewLimiter(rate.Inf, 0),
	}
	for range 10 {
		_, err := r.Request(context.Background(), "/test", nil, nil)
		require.NoError(t, err)
	}

	assert.Equal(t, 10, callCount)
}

func TestRequesterLimiter_RateLimited(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex

	called := false

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			mu.Lock()
			called = true
			mu.Unlock()

			return &resty.Response{}, nil
		},
	}
	limiter := rate.NewLimiter(1, 1)
	_ = limiter.Wait(context.Background())

	r := requesterLimiter{
		requester: inner,
		limiter:   limiter,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := r.Request(ctx, "/test", nil, nil)
	require.Error(t, err)

	mu.Lock()
	assert.False(t, called)
	mu.Unlock()
}

func TestRequesterLimiter_ConsecutiveRequestsWithBurst(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex

	callCount := 0

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			mu.Lock()
			callCount++
			mu.Unlock()

			return &resty.Response{}, nil
		},
	}

	limiter := rate.NewLimiter(100, 5)

	r := requesterLimiter{
		requester: inner,
		limiter:   limiter,
	}
	for range 5 {
		_, err := r.Request(context.Background(), "/test", nil, nil)
		require.NoError(t, err)
	}

	assert.Equal(t, 5, callCount)
}
