package tmdb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRequesterLazy_EmptyToken_ReturnsNilNil(t *testing.T) {
	t.Parallel()

	holder := NewAccessTokenHolder("")
	r := &requesterLazy{
		config:            Config{Enabled: true},
		accessTokenHolder: holder,
		logger:            zap.NewNop().Sugar(),
	}

	resp, err := r.Request(context.Background(), "/test", nil, nil)
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestNewRequester_EmptyToken_ReturnsNoop(t *testing.T) {
	t.Parallel()

	req, err := newRequester(context.Background(), Config{Enabled: true, AccessToken: ""}, zap.NewNop().Sugar())
	require.NoError(t, err)
	require.NotNil(t, req)

	resp, err := req.Request(context.Background(), "/test", nil, nil)
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestRequesterNoop_Request_ReturnsNilNil(t *testing.T) {
	t.Parallel()

	var n requesterNoop

	resp, err := n.Request(context.Background(), "/test", nil, nil)
	require.NoError(t, err)
	require.Nil(t, resp)
}
