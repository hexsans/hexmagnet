package httpserver

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewCors(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop().Sugar()
	cfg := NewDefaultConfig()

	result := NewCors(CorsParams{Config: cfg, Logger: logger})
	assert.NotNil(t, result.Option)
	assert.Equal(t, "cors", result.Option.Key())
}

func TestCorsOptionKey(t *testing.T) {
	t.Parallel()

	opt := corsOption{}
	assert.Equal(t, "cors", opt.Key())
}

func TestCorsOptionApply(t *testing.T) {
	t.Parallel()

	opt := corsOption{handlerFunc: func(_ *gin.Context) {}}
	g := gin.New()

	err := opt.Apply(g)

	assert.NoError(t, err)
}
