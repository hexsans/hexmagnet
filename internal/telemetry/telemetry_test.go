package telemetry

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_EmptyCollectors(t *testing.T) {
	t.Parallel()

	result, err := New(Params{})
	require.NoError(t, err)

	registry, regErr := result.Registry.Get()
	require.NoError(t, regErr)
	assert.NotNil(t, registry)
}

func TestNew_WithExtraCollector(t *testing.T) {
	t.Parallel()

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})

	result, err := New(Params{
		Collectors: []prometheus.Collector{counter},
	})
	require.NoError(t, err)

	registry, regErr := result.Registry.Get()
	require.NoError(t, regErr)
	assert.NotNil(t, registry)
}

func TestNew_DuplicatePanics(t *testing.T) {
	t.Parallel()

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})

	result, err := New(Params{
		Collectors: []prometheus.Collector{counter, counter},
	})
	require.NoError(t, err)

	_, regErr := result.Registry.Get()
	assert.Error(t, regErr)
}

func TestNew_NamespaceConstant(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "hexmagnet", Namespace)
}

func TestNew_LazyEvaluation(t *testing.T) {
	t.Parallel()

	result, err := New(Params{})
	require.NoError(t, err)

	registry, regErr := result.Registry.Get()
	require.NoError(t, regErr)
	assert.NotNil(t, registry)

	sameRegistry, sameErr := result.Registry.Get()
	require.NoError(t, sameErr)
	assert.Equal(t, registry, sameRegistry)
}
