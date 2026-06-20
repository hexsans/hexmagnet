package tmdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNew_Enabled_EmptyToken(t *testing.T) {
	t.Parallel()

	cfg := Config{Enabled: true, AccessToken: ""}
	p := Params{
		Config:            cfg,
		AccessTokenHolder: NewAccessTokenHolder(""),
		Logger:            zap.NewNop().Sugar(),
	}
	result := New(p)
	assert.NotNil(t, result.Client)
}

func TestNew_Disabled(t *testing.T) {
	t.Parallel()

	cfg := Config{Enabled: false, AccessToken: "token"}
	p := Params{
		Config:            cfg,
		AccessTokenHolder: NewAccessTokenHolder("token"),
		Logger:            zap.NewNop().Sugar(),
	}
	result := New(p)
	assert.NotNil(t, result.Client)
}

func TestNew_Enabled_WithToken(t *testing.T) {
	t.Parallel()

	cfg := Config{Enabled: true, AccessToken: "test-token"}
	p := Params{
		Config:            cfg,
		AccessTokenHolder: NewAccessTokenHolder("test-token"),
		Logger:            zap.NewNop().Sugar(),
	}
	result := New(p)
	assert.NotNil(t, result.Client)
}

func TestNew_Result_ClientLazy(t *testing.T) {
	t.Parallel()

	cfg := Config{Enabled: true, AccessToken: ""}
	p := Params{
		Config:            cfg,
		AccessTokenHolder: NewAccessTokenHolder(""),
		Logger:            zap.NewNop().Sugar(),
	}
	result := New(p)
	require.NotNil(t, result.Client)

	client, err := result.Client.Get()
	require.NoError(t, err)
	require.NotNil(t, client)
}
