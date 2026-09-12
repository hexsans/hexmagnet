package tmdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRestyLogger_AllLevelsAreDebug(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core).Sugar()

	l := restyLogger{logger: logger}
	l.Errorf("error %s", "x")
	l.Warnf("warn %s", "y")
	l.Debugf("debug %s", "z")

	require.Equal(t, 3, logs.Len())

	for _, entry := range logs.All() {
		assert.Equal(t, zap.DebugLevel, entry.Level)
	}

	assert.Equal(t, "error x", logs.All()[0].Message)
	assert.Equal(t, "warn y", logs.All()[1].Message)
	assert.Equal(t, "debug z", logs.All()[2].Message)
}
