package tmdb

import (
	"testing"

	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestNewHealthCheckFx(t *testing.T) {
	t.Parallel()

	cfg := Config{Enabled: true}
	lazy := utils.NewLazy(func() (Client, error) {
		return &mockClient{}, nil
	})
	p := HealthCheckFxParams{
		Config: cfg,
		Client: lazy,
	}
	result := NewHealthCheckFx(p)
	assert.NotNil(t, result.Option)
}

func TestNewHealthCheckFx_Disabled(t *testing.T) {
	t.Parallel()

	cfg := Config{Enabled: false}
	lazy := utils.NewLazy(func() (Client, error) {
		return &mockClient{}, nil
	})
	p := HealthCheckFxParams{
		Config: cfg,
		Client: lazy,
	}
	result := NewHealthCheckFx(p)
	assert.NotNil(t, result.Option)
}
