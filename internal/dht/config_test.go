package dht

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ZeroValues(t *testing.T) {
	t.Parallel()

	cfg := Config{}
	assert.Equal(t, uint(0), cfg.RescrapeThreshold)
}

func TestConfig_ExplicitValues(t *testing.T) {
	t.Parallel()

	cfg := Config{RescrapeThreshold: 3600}
	assert.Equal(t, uint(3600), cfg.RescrapeThreshold)
}
