package backoff

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelay(t *testing.T) {
	t.Parallel()

	cfg := Config{Base: time.Second, Factor: 2, Max: 10 * time.Second}

	assert.Equal(t, time.Second, cfg.Delay(1))
	assert.Equal(t, 2*time.Second, cfg.Delay(2))
	assert.Equal(t, 4*time.Second, cfg.Delay(3))
	assert.Equal(t, 8*time.Second, cfg.Delay(4))
	assert.Equal(t, 10*time.Second, cfg.Delay(5))
}

func TestDelay_ConstantFactor(t *testing.T) {
	t.Parallel()

	cfg := Config{Base: 5 * time.Second, Factor: 1, Max: time.Minute}

	assert.Equal(t, 5*time.Second, cfg.Delay(1))
	assert.Equal(t, 5*time.Second, cfg.Delay(10))
}

func TestDelay_Defaults(t *testing.T) {
	t.Parallel()

	cfg := Config{}

	assert.Equal(t, DefaultBase, cfg.Delay(1))
	assert.Equal(t, DefaultMax, cfg.Delay(100))
}

func TestWait(t *testing.T) {
	t.Parallel()

	cfg := Config{Base: time.Millisecond, Factor: 1, Max: time.Millisecond}

	require.NoError(t, cfg.Wait(context.Background(), 1))
}

func TestWait_Canceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Config{Base: time.Hour}.Wait(ctx, 1)
	require.ErrorIs(t, err, context.Canceled)
}
