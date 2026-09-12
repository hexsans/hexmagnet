package tmdb

import (
	"context"
	"errors"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequesterLogger_Success_LogsDebug(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core).Sugar()

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			return newRestyResponse(200), nil
		},
	}
	r := requesterLogger{
		requester: inner,
		logger:    logger,
	}
	_, err := r.Request(context.Background(), "/test/path", map[string]string{"key": "val"}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, logs.Len())
	assert.Equal(t, zap.DebugLevel, logs.All()[0].Level)
	assert.Contains(t, logs.All()[0].Message, "request succeeded")

	ctxMap := logs.All()[0].ContextMap()
	assert.Equal(t, "/test/path", ctxMap["path"])
	assert.Equal(t, map[string]string{"key": "val"}, ctxMap["queryParams"])
	assert.Contains(t, ctxMap, "status")
	assert.NotContains(t, ctxMap, "trace")
}

func TestRequesterLogger_Error_LogsWarn(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core).Sugar()

	expectedErr := errors.New("test error")
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			return nil, expectedErr
		},
	}
	r := requesterLogger{
		requester: inner,
		logger:    logger,
	}
	_, err := r.Request(context.Background(), "/test/path", nil, nil)
	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, logs.Len())
	assert.Equal(t, zap.WarnLevel, logs.All()[0].Level)
	assert.Contains(t, logs.All()[0].Message, "request failed")

	ctxMap := logs.All()[0].ContextMap()
	assert.Equal(t, "/test/path", ctxMap["path"])
	assert.Contains(t, ctxMap, "error")
}

func TestRequesterLogger_NilResponse_LogsWarn(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core).Sugar()

	expectedErr := errors.New("nil response error")
	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			return nil, expectedErr
		},
	}
	r := requesterLogger{
		requester: inner,
		logger:    logger,
	}
	_, err := r.Request(context.Background(), "/path", nil, nil)
	require.Error(t, err)
	assert.Equal(t, 1, logs.Len())
	assert.Equal(t, zap.WarnLevel, logs.All()[0].Level)
}

func TestRequesterLogger_ErrorStatus_LogsWarn(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core).Sugar()

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			return newRestyResponse(500), errors.New("server error")
		},
	}
	r := requesterLogger{
		requester: inner,
		logger:    logger,
	}
	_, err := r.Request(context.Background(), "/path", nil, nil)
	require.Error(t, err)
	assert.Equal(t, 1, logs.Len())
	assert.Equal(t, zap.WarnLevel, logs.All()[0].Level)
}

func TestRequesterLogger_EmptyQueryParams(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core).Sugar()

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			return newRestyResponse(200), nil
		},
	}
	r := requesterLogger{
		requester: inner,
		logger:    logger,
	}
	_, err := r.Request(context.Background(), "/path", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, logs.Len())

	ctxMap := logs.All()[0].ContextMap()
	assert.Nil(t, ctxMap["queryParams"])
}

func TestRequesterLogger_DifferentPaths(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core).Sugar()

	inner := &mockRequester{
		fn: func(_ context.Context, _ string, _ map[string]string, _ any) (*resty.Response, error) {
			return newRestyResponse(200), nil
		},
	}
	r := requesterLogger{
		requester: inner,
		logger:    logger,
	}

	_, err := r.Request(context.Background(), "/path1", nil, nil)
	require.NoError(t, err)
	_, err = r.Request(context.Background(), "/path2", nil, nil)
	require.NoError(t, err)

	assert.Equal(t, 2, logs.Len())
	assert.Equal(t, "/path1", logs.All()[0].ContextMap()["path"])
	assert.Equal(t, "/path2", logs.All()[1].ContextMap()["path"])
}
