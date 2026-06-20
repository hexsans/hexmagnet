package tmdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	assert.True(t, cfg.Enabled)
	assert.Empty(t, cfg.AccessToken)
	assert.Equal(t, 20, cfg.RateLimit)
}
