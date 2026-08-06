package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	assert.Equal(t, []string{defaultAddress}, cfg.Addresses)

	// Verify immutability
	cfg2 := DefaultConfig()
	cfg2.Addresses[0] = "http://other:9200"
	assert.Equal(t, []string{defaultAddress}, DefaultConfig().Addresses)
}
