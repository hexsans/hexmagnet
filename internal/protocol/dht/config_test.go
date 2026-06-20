package dht

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()

	assert.Equal(t, uint16(3334), cfg.Port)
	assert.True(t, cfg.Responder.Enabled)
	assert.Equal(t, 50, cfg.Responder.GlobalRateLimit)
	assert.Equal(t, 1, cfg.Responder.PerIPRateLimit)
	assert.Equal(t, time.Minute, cfg.ReseedBootstrapNodesInterval)
	assert.NotEmpty(t, cfg.BootstrapNodes)

	expectedNodes := []string{
		"router.utorrent.com:6881",
		"router.bittorrent.com:6881",
		"dht.transmissionbt.com:6881",
		"dht.aelitis.com:6881",
		"router.silotis.us:6881",
		"dht.libtorrent.org:25401",
	}
	assert.Equal(t, expectedNodes, cfg.BootstrapNodes)
}
