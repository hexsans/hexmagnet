package httpserver

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type optionMock struct {
	key string
}

func (o optionMock) Key() string           { return o.key }
func (optionMock) Apply(*gin.Engine) error { return nil }

func TestNew(t *testing.T) {
	t.Parallel()

	p := Params{
		Config:       NewDefaultConfig(),
		Options:      []Option{},
		Logger:       zap.NewNop(),
		ServerConfig: servercfg.NewDefaultConfig(),
	}
	result := New(p)

	assert.NotNil(t, result.Worker)
	assert.Equal(t, "http_server", result.Worker.Key())
}

func TestResolveOptions(t *testing.T) {
	t.Parallel()

	optA := optionMock{key: "a"}
	optB := optionMock{key: "b"}
	optC := optionMock{key: "c"}

	t.Run("empty params and options", func(t *testing.T) {
		t.Parallel()

		result, err := resolveOptions([]string{}, []Option{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("star enables all sorted by key", func(t *testing.T) {
		t.Parallel()

		result, err := resolveOptions([]string{"*"}, []Option{optC, optA, optB})
		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, "a", result[0].Key())
		assert.Equal(t, "b", result[1].Key())
		assert.Equal(t, "c", result[2].Key())
	})

	t.Run("specific key enables only that option", func(t *testing.T) {
		t.Parallel()

		result, err := resolveOptions([]string{"b"}, []Option{optA, optB, optC})
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "b", result[0].Key())
	})

	t.Run("no matching param means no options enabled", func(t *testing.T) {
		t.Parallel()

		result, err := resolveOptions([]string{}, []Option{optA, optB})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("duplicate option key errors", func(t *testing.T) {
		t.Parallel()

		_, err := resolveOptions([]string{"*"}, []Option{optA, optA})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("unknown param key errors", func(t *testing.T) {
		t.Parallel()

		_, err := resolveOptions([]string{"unknown"}, []Option{optA})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})
}
