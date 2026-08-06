package tmdb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewRequester_Disabled(t *testing.T) {
	t.Parallel()

	req, err := newRequester(context.Background(), Config{Enabled: false, AccessToken: ""}, zap.NewNop().Sugar())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
	assert.Nil(t, req)
}

func TestRequesterLazy_Disabled_ReturnsError(t *testing.T) {
	t.Parallel()

	holder := NewAccessTokenHolder("some-token")
	r := &requesterLazy{
		config:            Config{Enabled: false, AccessToken: "some-token"},
		accessTokenHolder: holder,
		logger:            zap.NewNop().Sugar(),
	}
	_, err := r.Request(context.Background(), "/test", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}
