package tmdb

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccessTokenHolder(t *testing.T) {
	t.Parallel()

	h := NewAccessTokenHolder("test-token")
	require.NotNil(t, h)
	assert.Equal(t, "test-token", h.Get())
}

func TestNewAccessTokenHolder_Empty(t *testing.T) {
	t.Parallel()

	h := NewAccessTokenHolder("")
	require.NotNil(t, h)
	assert.Empty(t, h.Get())
}

func TestAccessTokenHolder_Set(t *testing.T) {
	t.Parallel()

	h := NewAccessTokenHolder("old")
	h.Set("new")
	assert.Equal(t, "new", h.Get())
}

func TestAccessTokenHolder_Set_Empty(t *testing.T) {
	t.Parallel()

	h := NewAccessTokenHolder("token")
	h.Set("")
	assert.Empty(t, h.Get())
}

func TestAccessTokenHolder_Set_Get_AfterSet(t *testing.T) {
	t.Parallel()

	h := NewAccessTokenHolder("initial")
	h.Set("updated")
	assert.Equal(t, "updated", h.Get())
	h.Set("final")
	assert.Equal(t, "final", h.Get())
}

func TestAccessTokenHolder_Concurrent(t *testing.T) {
	t.Parallel()

	h := NewAccessTokenHolder("initial")

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_ = h.Get()
			h.Set("new-token")
			_ = h.Get()
		}()
	}

	wg.Wait()
	assert.Equal(t, "new-token", h.Get())
}
