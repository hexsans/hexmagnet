package dbsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFromPool(t *testing.T) {
	t.Parallel()

	s := NewFromPool(nil)
	assert.NotNil(t, s)
	_, ok := s.(*pgSearch)
	assert.True(t, ok)
}
