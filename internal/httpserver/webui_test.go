package httpserver

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewWebUI(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop().Sugar()
	p := WebUIParams{Logger: logger}

	result := NewWebUI(p)
	assert.NotNil(t, result.Option)
	assert.Equal(t, "webui", result.Option.Key())
}

func TestBuilderKey(t *testing.T) {
	t.Parallel()

	b := &builder{logger: zap.NewNop().Sugar()}
	assert.Equal(t, "webui", b.Key())
}

func TestBuilderApply(t *testing.T) {
	t.Parallel()

	b := &builder{logger: zap.NewNop().Sugar()}
	g := gin.New()

	err := b.Apply(g)
	require.NoError(t, err)

	routes := g.Routes()
	assert.NotEmpty(t, routes)
}
