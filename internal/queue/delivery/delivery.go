// Package delivery defines the at-least-once delivery contract shared by the
// queue backends.
//
// A message handler that returns a permanent error (wrapped with
// permanent.Mark) acknowledges the message and drops it. Any other error
// leaves the message unacknowledged so it is redelivered with exponential
// backoff. When a message exceeds MaxAttempts it is treated as poison:
// discarded with an error log so it cannot block the queue forever.
package delivery

import (
	"time"

	"github.com/hexsans/hexmagnet/internal/backoff"
)

const (
	DefaultMaxAttempts = 5
	DefaultBackoffBase = time.Second
	DefaultBackoffMax  = 30 * time.Second
)

type Config struct {
	MaxAttempts int
	Backoff     backoff.Config
}

func DefaultConfig() Config {
	return Config{
		MaxAttempts: DefaultMaxAttempts,
		Backoff: backoff.Config{
			Base:   DefaultBackoffBase,
			Factor: 2,
			Max:    DefaultBackoffMax,
		},
	}
}

// Normalize fills zero-valued fields with their defaults, so callers can rely
// on a usable policy even when delivery was never configured explicitly.
func (c Config) Normalize() Config {
	defaults := DefaultConfig()

	if c.MaxAttempts <= 0 {
		c.MaxAttempts = defaults.MaxAttempts
	}

	if c.Backoff.Base <= 0 {
		c.Backoff.Base = defaults.Backoff.Base
	}

	if c.Backoff.Factor < 1 {
		c.Backoff.Factor = defaults.Backoff.Factor
	}

	if c.Backoff.Max <= 0 {
		c.Backoff.Max = defaults.Backoff.Max
	}

	return c
}
