package tmdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHealthCheck(t *testing.T) {
	t.Parallel()

	enabled := true
	lazy := utils.NewLazy(func() (Client, error) {
		return &mockClient{}, nil
	})
	hc := NewHealthCheck(enabled, lazy)
	assert.Equal(t, "tmdb", hc.Name)
	assert.Equal(t, 30*time.Second, hc.Timeout)
	assert.True(t, hc.IsActive())
}

func TestNewHealthCheck_NotActive(t *testing.T) {
	t.Parallel()

	hc := NewHealthCheck(false, utils.NewLazy(func() (Client, error) {
		return &mockClient{}, nil
	}))
	assert.False(t, hc.IsActive())
}

func TestNewHealthCheck_Check_Success(t *testing.T) {
	t.Parallel()

	lazy := utils.NewLazy(func() (Client, error) {
		return &mockClient{}, nil
	})
	hc := NewHealthCheck(true, lazy)
	err := hc.Check(context.Background())
	assert.NoError(t, err)
}

func TestNewHealthCheck_Check_LazyError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("lazy init error")
	lazy := utils.NewLazy(func() (Client, error) {
		return nil, expectedErr
	})
	hc := NewHealthCheck(true, lazy)
	err := hc.Check(context.Background())
	assert.ErrorIs(t, err, expectedErr)
}

func TestNewHealthCheck_Check_ValidateError(t *testing.T) {
	t.Parallel()

	lazy := utils.NewLazy(func() (Client, error) {
		return &mockClient{validateErr: ErrUnauthorized}, nil
	})
	hc := NewHealthCheck(true, lazy)
	err := hc.Check(context.Background())
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestNewHealthCheck_Params(t *testing.T) {
	t.Parallel()

	lazy := utils.NewLazy(func() (Client, error) {
		return &mockClient{}, nil
	})
	hc := NewHealthCheck(true, lazy)
	require.NotNil(t, hc)
	assert.Equal(t, "tmdb", hc.Name)
}
