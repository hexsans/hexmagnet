package httpserver

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrometheusBuilder_Key(t *testing.T) {
	t.Parallel()

	b := prometheusBuilder{}
	assert.Equal(t, "prometheus", b.Key())
}

func TestPrometheusBuilder_Apply_Error(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	reg := utils.NewLazy(func() (*prometheus.Registry, error) {
		return nil, assert.AnError
	})
	b := prometheusBuilder{registry: reg}
	engine := gin.New()
	err := b.Apply(engine)
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestPrometheusBuilder_Apply_Success(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	r := prometheus.NewRegistry()
	reg := utils.NewLazy(func() (*prometheus.Registry, error) {
		return r, nil
	})
	b := prometheusBuilder{registry: reg}
	engine := gin.New()
	err := b.Apply(engine)
	require.NoError(t, err)

	assert.Len(t, engine.Routes(), 9)
	assert.Equal(t, "/metrics", engine.Routes()[0].Path)
}
