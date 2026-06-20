package httpserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()

	assert.Equal(t, "release", cfg.GinMode)
	assert.Equal(t, 10*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 60*time.Second, cfg.IdleTimeout)
	assert.Equal(t, []string{"*"}, cfg.Options)

	assert.Equal(t, []string(nil), cfg.Cors.AllowedOrigins)
	assert.Equal(t, []string(nil), cfg.Cors.AllowedMethods)
	assert.Equal(t, []string(nil), cfg.Cors.AllowedHeaders)
	assert.Equal(t, []string(nil), cfg.Cors.ExposedHeaders)
	assert.Equal(t, 0, cfg.Cors.MaxAge)
	assert.False(t, cfg.Cors.AllowCredentials)
	assert.False(t, cfg.Cors.AllowPrivateNetwork)
	assert.False(t, cfg.Cors.OptionsPassthrough)
	assert.Equal(t, 0, cfg.Cors.OptionsSuccessStatus)
	assert.False(t, cfg.Cors.Debug)
}
