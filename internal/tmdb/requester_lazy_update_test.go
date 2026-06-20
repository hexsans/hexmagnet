package tmdb

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRequesterLazy_UpdateConfig_AppliesTokenAndResetsRequester(t *testing.T) {
	t.Parallel()

	holder := NewAccessTokenHolder("old-token")
	r := &requesterLazy{
		config:            Config{Enabled: true, AccessToken: "old-token", RateLimit: 20},
		requester:         requesterNoop{},
		lastToken:         "old-token",
		accessTokenHolder: holder,
		logger:            zap.NewNop().Sugar(),
	}

	r.UpdateConfig(Config{Enabled: true, AccessToken: "new-token", RateLimit: 5})

	require.Equal(t, "new-token", holder.Get())

	r.mu.Lock()
	defer r.mu.Unlock()

	require.Nil(t, r.requester)
	require.Empty(t, r.lastToken)
	require.Equal(t, 5, r.config.RateLimit)
}

func TestRequesterLazy_UpdateConfig_NoopOnSameConfig(t *testing.T) {
	t.Parallel()

	holder := NewAccessTokenHolder("token")
	r := &requesterLazy{
		config:            Config{Enabled: true, AccessToken: "token", RateLimit: 20},
		requester:         requesterNoop{},
		lastToken:         "token",
		accessTokenHolder: holder,
		logger:            zap.NewNop().Sugar(),
	}

	built := r.requester

	r.UpdateConfig(Config{Enabled: true, AccessToken: "token", RateLimit: 20})

	r.mu.Lock()
	defer r.mu.Unlock()

	require.Equal(t, built, r.requester)
	require.Equal(t, "token", r.lastToken)
}

func TestRequesterLazy_UpdateConfig_DisableClearsRequester(t *testing.T) {
	t.Parallel()

	holder := NewAccessTokenHolder("token")
	r := &requesterLazy{
		config:            Config{Enabled: true, AccessToken: "token", RateLimit: 20},
		requester:         requesterNoop{},
		lastToken:         "token",
		accessTokenHolder: holder,
		logger:            zap.NewNop().Sugar(),
	}

	r.UpdateConfig(Config{Enabled: false, AccessToken: "token", RateLimit: 20})

	r.mu.Lock()
	require.Nil(t, r.requester)
	r.mu.Unlock()
}
