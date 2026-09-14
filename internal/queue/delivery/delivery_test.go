package delivery

import (
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/backoff"
	"github.com/stretchr/testify/assert"
)

func TestNormalize_Defaults(t *testing.T) {
	t.Parallel()

	cfg := Config{}.Normalize()
	assert.Equal(t, DefaultMaxAttempts, cfg.MaxAttempts)
	assert.Equal(t, DefaultBackoffBase, cfg.Backoff.Base)
	assert.InDelta(t, 2.0, cfg.Backoff.Factor, 0)
	assert.Equal(t, DefaultBackoffMax, cfg.Backoff.Max)
}

func TestNormalize_KeepsExplicitValues(t *testing.T) {
	t.Parallel()

	cfg := Config{
		MaxAttempts: 2,
		Backoff: backoff.Config{
			Base:   time.Millisecond,
			Factor: 3,
			Max:    time.Second,
		},
	}.Normalize()

	assert.Equal(t, 2, cfg.MaxAttempts)
	assert.Equal(t, time.Millisecond, cfg.Backoff.Base)
	assert.InDelta(t, 3.0, cfg.Backoff.Factor, 0)
	assert.Equal(t, time.Second, cfg.Backoff.Max)
}
