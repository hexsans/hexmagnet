package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithKey(t *testing.T) {
	t.Parallel()

	r := &baseResolver{}
	r.applyOptions(WithKey("custom-key"))
	assert.Equal(t, "custom-key", r.Key())
}

func TestWithPriority(t *testing.T) {
	t.Parallel()

	r := &baseResolver{}
	r.applyOptions(WithPriority(99))
	assert.Equal(t, 99, r.Priority())
}

func TestWithKeyAndPriority(t *testing.T) {
	t.Parallel()

	r := &baseResolver{}
	r.applyOptions(WithKey("k"), WithPriority(5))
	assert.Equal(t, "k", r.Key())
	assert.Equal(t, 5, r.Priority())
}
