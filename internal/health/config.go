package health

import (
	"context"
	"time"
)

type (
	// Check allows to configure health checks.
	Check struct {
		// The Name must be unique among all checks. Name is a required attribute.
		Name string // Required

		IsActive func() bool

		// Check is the check function that will be executed to check availability.
		// This function must return an error if the checked service is considered
		// not available. Check is a required attribute.
		Check func(ctx context.Context) error // Required

		// Timeout will override the global timeout value, if it is smaller than
		// the global timeout (see WithTimeout).
		Timeout time.Duration // Optional

		// MaxTimeInError will set a duration for how long a service must be
		// in an error state until it is considered down/unavailable.
		MaxTimeInError time.Duration // Optional

		// MaxContiguousFails will set a maximum number of contiguous
		// check fails until the service is considered down/unavailable.
		MaxContiguousFails uint // Optional

		// StatusListener allows to set a listener that will be called
		// whenever the AvailabilityStatus (e.g. from "up" to "down").
		StatusListener func(ctx context.Context, name string, state CheckState) // Optional

		// Interceptors holds a list of Interceptor instances that will be executed one after another in the
		// order as they appear in the list.
		Interceptors []Interceptor

		// DisablePanicRecovery disables automatic recovery from panics. If left in its default value (false),
		// panics will be automatically converted into errors instead.
		DisablePanicRecovery bool

		updateInterval time.Duration
		initialDelay   time.Duration
	}

	// CheckerOption is a configuration option for a Checker.
	CheckerOption func(config *checkerConfig)

	// HandlerOption is a configuration option for a Handler (see NewHandler).
	HandlerOption func(*HandlerConfig)
)

// NewChecker creates a new Checker. The provided options will be
// used to modify its configuration. If the Checker was not yet started
// (see Checker.IsStarted), it will be started automatically
// (see Checker.Start). You can disable this autostart by
// adding the WithDisabledAutostart configuration option.
func NewChecker(options ...CheckerOption) Checker {
	cfg := checkerConfig{
		cacheTTL:     1 * time.Second,
		timeout:      10 * time.Second,
		checks:       map[string]*Check{},
		interceptors: []Interceptor{},
	}

	for _, opt := range options {
		opt(&cfg)
	}

	return newChecker(cfg)
}

// WithPeriodicCheck adds a new health check that contributes to the overall service availability status.
// The health check will be performed on a fixed schedule and will not be executed for each HTTP request
// (as in contrast to WithCheck). This allows to process a much higher number of HTTP requests without
// actually calling the checked services too often or to execute long-running checks.
// This way Checker.Check (and the health endpoint) always returns the last result of the periodic check.
func WithPeriodicCheck(refreshPeriod time.Duration, initialDelay time.Duration, check Check) CheckerOption {
	return func(cfg *checkerConfig) {
		check.updateInterval = refreshPeriod
		check.initialDelay = initialDelay
		cfg.checks[check.Name] = &check
	}
}

// WithInfo sets values that will be available in every health check result. For example, you can use this option
// if you want to set information about your system that will be returned in every health check result, such as
// version number, Git SHA, build date, etc. These values will be available in CheckerResult.Info. If you use
// the default HTTP handler of this library (see NewHandler) or convert the CheckerResult to JSON on your own,
// these values will be available in the "info" field.
func WithInfo(values map[string]any) CheckerOption {
	return func(cfg *checkerConfig) {
		cfg.info = values
	}
}

// WithStatusChangeListener registers a listener that is called whenever the
// aggregated availability status changes (e.g. from "up" to "down").
func WithStatusChangeListener(fn func(context.Context, CheckerState)) CheckerOption {
	return func(cfg *checkerConfig) {
		cfg.statusChangeListener = fn
	}
}
