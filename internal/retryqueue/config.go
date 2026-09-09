package retryqueue

import (
	"math"
	"time"
)

// Config controls the torrent retry queue behaviour.
type Config struct {
	Enabled bool `yaml:"enabled"`

	// MaxRetries is the maximum number of retry attempts per torrent before
	// it gets deleted. A negative value means "always retry".
	MaxRetries int `yaml:"max_retries"`

	// Interval is the base delay before the first retry.
	Interval time.Duration `yaml:"interval"`

	// BackoffFactor is the multiplier applied to the delay on every
	// subsequent failure (exponential backoff).
	BackoffFactor float64 `yaml:"backoff_factor"`

	// MaxInterval caps the exponential backoff delay.
	MaxInterval time.Duration `yaml:"max_interval"`

	// ScanInterval is how often the scheduler scans for due entries.
	ScanInterval time.Duration `yaml:"scan_interval"`

	// BatchSize is the maximum number of due entries dispatched per scan.
	BatchSize int `yaml:"batch_size"`

	// DispatchLease is how long a dispatched entry is held before it
	// becomes due again if the retry outcome is never reported.
	DispatchLease time.Duration `yaml:"dispatch_lease"`
}

func NewDefaultConfig() Config {
	return Config{
		Enabled:       true,
		MaxRetries:    3,
		Interval:      5 * time.Minute,
		BackoffFactor: 2.0,
		MaxInterval:   time.Hour,
		ScanInterval:  30 * time.Second,
		BatchSize:     100,
		DispatchLease: 15 * time.Minute,
	}
}

func (c Config) backoffDelay(failCount int32) time.Duration {
	interval := c.Interval
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	factor := c.BackoffFactor
	if factor <= 1 {
		factor = 1
	}

	maxInterval := c.MaxInterval
	if maxInterval <= 0 {
		maxInterval = time.Hour
	}

	delay := time.Duration(float64(interval) * math.Pow(factor, float64(failCount-1)))
	if delay <= 0 || delay > maxInterval {
		return maxInterval
	}

	return delay
}
