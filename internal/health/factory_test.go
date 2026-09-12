package health

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestStatusLogger_LogsTransitions(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core).Sugar()

	listener := statusLogger(logger)

	listener(context.Background(), CheckerState{Status: StatusUp})
	assert.Zero(t, logs.Len(), "initial up must not log")

	listener(context.Background(), CheckerState{
		Status: StatusDown,
		CheckState: map[string]CheckState{
			"postgres": {Status: StatusDown, Result: errors.New("ping failed")},
			"tmdb":     {Status: StatusUp},
		},
	})

	listener(context.Background(), CheckerState{Status: StatusUp})

	require.Equal(t, 2, logs.Len())

	degraded := logs.All()[0]
	assert.Equal(t, zap.ErrorLevel, degraded.Level)
	assert.Equal(t, "service health degraded", degraded.Message)
	assert.Contains(t, degraded.ContextMap()["failing"], "postgres: ping failed")

	recovered := logs.All()[1]
	assert.Equal(t, zap.InfoLevel, recovered.Level)
	assert.Equal(t, "service health recovered", recovered.Message)
}

func TestStatusLogger_FirstStatusDownLogsError(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core).Sugar()

	listener := statusLogger(logger)

	listener(context.Background(), CheckerState{Status: StatusDown})

	require.Equal(t, 1, logs.Len())
	assert.Equal(t, zap.ErrorLevel, logs.All()[0].Level)
}
