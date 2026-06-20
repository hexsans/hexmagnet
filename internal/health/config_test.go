package health

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithPeriodicCheckConfig(t *testing.T) {
	t.Parallel()

	// Arrange
	expectedName := "test"
	cfg := checkerConfig{checks: map[string]*Check{}}
	interval := 5 * time.Second
	initialDelay := 1 * time.Minute
	check := Check{Name: expectedName, updateInterval: interval, initialDelay: initialDelay}

	// Act
	WithPeriodicCheck(interval, initialDelay, check)(&cfg)

	// Assert
	assert.Len(t, cfg.checks, 1)
	assert.True(t, reflect.DeepEqual(check, *cfg.checks[expectedName]))
}

func TestNewWithDefaults(t *testing.T) {
	t.Parallel()

	// Arrange
	configApplied := false
	opt := func(*checkerConfig) { configApplied = true }

	// Act
	checker := NewChecker(opt)

	// Assert
	ckr := checker.(*defaultChecker)
	assert.Equal(t, 1*time.Second, ckr.cfg.cacheTTL)
	assert.Equal(t, 10*time.Second, ckr.cfg.timeout)
	assert.True(t, configApplied)
}

func TestCheckerAutostartConfig(t *testing.T) {
	t.Parallel()

	// Arrange + Act
	c := NewChecker()

	// Assert
	assert.True(t, c.IsStarted())
}
