package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestKeyedLimiter_Allow_Unlimited(t *testing.T) {
	t.Parallel()

	lim := NewKeyedLimiter(rate.Inf, 1, 100, time.Minute)
	assert.True(t, lim.Allow("key1"))
	assert.True(t, lim.Allow("key1"))
	assert.True(t, lim.Allow("key1"))
}

func TestKeyedLimiter_Allow_Limited(t *testing.T) {
	t.Parallel()

	lim := NewKeyedLimiter(1, 1, 100, time.Minute)
	assert.True(t, lim.Allow("key1"))
	assert.False(t, lim.Allow("key1"))
}

func TestKeyedLimiter_Allow_Burst(t *testing.T) {
	t.Parallel()

	lim := NewKeyedLimiter(1, 3, 100, time.Minute)
	assert.True(t, lim.Allow("key1"))
	assert.True(t, lim.Allow("key1"))
	assert.True(t, lim.Allow("key1"))
	assert.False(t, lim.Allow("key1"))
}

func TestKeyedLimiter_DifferentKeys(t *testing.T) {
	t.Parallel()

	lim := NewKeyedLimiter(1, 1, 100, time.Minute)
	assert.True(t, lim.Allow("key1"))
	assert.False(t, lim.Allow("key1"))
	assert.True(t, lim.Allow("key2"))
	assert.False(t, lim.Allow("key2"))
}

func TestKeyedLimiter_Wait_Unlimited(t *testing.T) {
	t.Parallel()

	lim := NewKeyedLimiter(rate.Inf, 1, 100, time.Minute)
	ctx := context.Background()

	err := lim.Wait(ctx, "key1")
	require.NoError(t, err)

	err = lim.Wait(ctx, "key1")
	require.NoError(t, err)
}

func TestKeyedLimiter_Wait_ContextCancel(t *testing.T) {
	t.Parallel()

	lim := NewKeyedLimiter(1, 1, 100, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_ = lim.Allow("key1")

	err := lim.Wait(ctx, "key1")
	assert.Error(t, err)
}

func TestKeyedLimiter_LRUEviction_RecreatesLimiter(t *testing.T) {
	t.Parallel()

	lim := NewKeyedLimiter(1, 1, 2, time.Minute)

	assert.True(t, lim.Allow("key1"))
	assert.True(t, lim.Allow("key2"))
	assert.True(t, lim.Allow("key3"))

	assert.True(t, lim.Allow("key4"))
}

func TestKeyedLimiter_Wait_Blocks(t *testing.T) {
	t.Parallel()

	lim := NewKeyedLimiter(100, 1, 100, time.Minute)

	ctx := context.Background()
	_ = lim.Allow("key1")

	done := make(chan error, 1)
	go func() {
		done <- lim.Wait(ctx, "key1")
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Wait should have returned within 500ms at rate 100/s")
	}
}
