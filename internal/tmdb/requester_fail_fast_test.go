package tmdb

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequesterFailFast_FirstCallSucceeds(t *testing.T) {
	t.Parallel()

	called := false
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			called = true
			return &resty.Response{}, nil
		},
	}
	r := requesterFailFast{
		requester:      inner,
		isUnauthorized: &concurrency.AtomicValue[bool]{},
	}
	_, err := r.Request(context.Background(), "/test", nil, nil)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRequesterFailFast_SecondCallPassesAfterSuccess(t *testing.T) {
	t.Parallel()

	callCount := 0
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			callCount++
			return &resty.Response{}, nil
		},
	}
	r := requesterFailFast{
		requester:      inner,
		isUnauthorized: &concurrency.AtomicValue[bool]{},
	}

	_, err := r.Request(context.Background(), "/test", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	_, err = r.Request(context.Background(), "/test", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestRequesterFailFast_Unauthorized_FailsFast(t *testing.T) {
	t.Parallel()

	callCount := 0
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			callCount++
			return nil, ErrUnauthorized
		},
	}
	r := requesterFailFast{
		requester:      inner,
		isUnauthorized: &concurrency.AtomicValue[bool]{},
	}

	_, err := r.Request(context.Background(), "/test", nil, nil)
	require.ErrorIs(t, err, ErrUnauthorized)
	assert.Equal(t, 1, callCount)

	_, err = r.Request(context.Background(), "/test", nil, nil)
	require.ErrorIs(t, err, ErrUnauthorized)
	assert.Equal(t, 1, callCount)
}

func TestRequesterFailFast_Unauthorized_AllSubsequentFailFast(t *testing.T) {
	t.Parallel()

	callCount := 0
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			callCount++
			return nil, ErrUnauthorized
		},
	}
	r := requesterFailFast{
		requester:      inner,
		isUnauthorized: &concurrency.AtomicValue[bool]{},
	}

	for range 5 {
		_, err := r.Request(context.Background(), "/test", nil, nil)
		require.ErrorIs(t, err, ErrUnauthorized)
	}

	assert.Equal(t, 1, callCount)
}

func TestRequesterFailFast_OtherError_DoesNotFailFast(t *testing.T) {
	t.Parallel()

	callCount := 0
	otherErr := errors.New("some other error")
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			callCount++
			return nil, otherErr
		},
	}
	r := requesterFailFast{
		requester:      inner,
		isUnauthorized: &concurrency.AtomicValue[bool]{},
	}

	_, err := r.Request(context.Background(), "/test", nil, nil)
	require.ErrorIs(t, err, otherErr)
	assert.Equal(t, 1, callCount)

	_, err = r.Request(context.Background(), "/test", nil, nil)
	require.ErrorIs(t, err, otherErr)
	assert.Equal(t, 2, callCount)
}

func TestRequesterFailFast_SuccessThenUnauthorized_FailsFast(t *testing.T) {
	t.Parallel()

	callCount := 0
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			callCount++
			if callCount == 1 {
				return &resty.Response{}, nil
			}

			return nil, ErrUnauthorized
		},
	}
	r := requesterFailFast{
		requester:      inner,
		isUnauthorized: &concurrency.AtomicValue[bool]{},
	}

	_, err := r.Request(context.Background(), "/test", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	_, err = r.Request(context.Background(), "/test", nil, nil)
	require.ErrorIs(t, err, ErrUnauthorized)
	assert.Equal(t, 2, callCount)

	_, err = r.Request(context.Background(), "/test", nil, nil)
	require.ErrorIs(t, err, ErrUnauthorized)
	assert.Equal(t, 2, callCount)
}

func TestRequesterFailFast_UnauthorizedWrapped_FailsFast(t *testing.T) {
	t.Parallel()

	callCount := 0
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			callCount++
			return nil, fmt.Errorf("wrapped: %w", ErrUnauthorized)
		},
	}
	r := requesterFailFast{
		requester:      inner,
		isUnauthorized: &concurrency.AtomicValue[bool]{},
	}

	_, err := r.Request(context.Background(), "/test", nil, nil)
	require.ErrorIs(t, err, ErrUnauthorized)
	assert.Equal(t, 1, callCount)

	_, err = r.Request(context.Background(), "/test", nil, nil)
	require.ErrorIs(t, err, ErrUnauthorized)
	assert.Equal(t, 1, callCount)
}

func TestRequesterFailFast_NilResponse_NotTreatedAsUnauthorized(t *testing.T) {
	t.Parallel()

	callCount := 0
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			callCount++
			return nil, errors.New("unexpected nil")
		},
	}
	r := requesterFailFast{
		requester:      inner,
		isUnauthorized: &concurrency.AtomicValue[bool]{},
	}

	for range 3 {
		_, err := r.Request(context.Background(), "/test", nil, nil)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrUnauthorized)
	}

	assert.Equal(t, 3, callCount)
}
