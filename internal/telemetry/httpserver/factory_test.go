//go:build !debug

package httpserver

import (
	"testing"

	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	reg := utils.NewLazy(func() (*prometheus.Registry, error) {
		return prometheus.NewRegistry(), nil
	})
	p := Params{PrometheusRegistry: reg}
	result := New(p)
	assert.NotNil(t, result.PrometheusOption)
}

func TestNew_GetError_Propagation(t *testing.T) {
	t.Parallel()

	reg := utils.NewLazy(func() (*prometheus.Registry, error) {
		return nil, assert.AnError
	})
	p := Params{PrometheusRegistry: reg}
	result := New(p)
	assert.NotNil(t, result.PrometheusOption)
}
