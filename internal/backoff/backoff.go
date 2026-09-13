// Package backoff provides shared retry-delay helpers used by the queue,
// webhook and crawler components.
package backoff

import (
	"context"
	"math"
	"time"
)

const (
	DefaultBase   = time.Second
	DefaultFactor = 2.0
	DefaultMax    = time.Minute
)

// Config describes an exponential backoff schedule. Base is the delay before
// the first retry, Factor multiplies the delay on every subsequent retry, and
// Max caps the delay. Invalid values fall back to defaults.
type Config struct {
	Base   time.Duration
	Factor float64
	Max    time.Duration
}

// Delay returns the delay before the given retry attempt. Attempts start at 1.
func (c Config) Delay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	base := c.Base
	if base <= 0 {
		base = DefaultBase
	}

	factor := c.Factor
	if factor < 1 {
		factor = DefaultFactor
	}

	maxDelay := c.Max
	if maxDelay <= 0 {
		maxDelay = DefaultMax
	}

	delay := time.Duration(float64(base) * math.Pow(factor, float64(attempt-1)))
	if delay <= 0 || delay > maxDelay {
		return maxDelay
	}

	return delay
}

// Wait sleeps for the delay before the given retry attempt. It returns early
// with ctx.Err() when the context is canceled.
func (c Config) Wait(ctx context.Context, attempt int) error {
	timer := time.NewTimer(c.Delay(attempt))
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
