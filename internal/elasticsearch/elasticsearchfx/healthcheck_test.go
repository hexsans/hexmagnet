package elasticsearchfx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func TestNewHealthCheckFx_AlwaysRegisters(t *testing.T) {
	t.Parallel()

	cfg := concurrency.AtomicValue[indexer.SearchConfig]{}
	cfg.Set(indexer.SearchConfig{Backend: backendPostgresql})

	result := newHealthCheckFx(healthCheckFxParams{SearchConfig: &cfg})

	assert.NotNil(t, result.Option)

	checker := health.NewChecker(result.Option)
	assert.NotNil(t, checker)
	checker.Stop()
}

func TestNewHealthCheckFx_CollectedInFxValueGroup(t *testing.T) {
	t.Parallel()

	cfg := concurrency.AtomicValue[indexer.SearchConfig]{}
	cfg.Set(indexer.SearchConfig{Backend: backendPostgresql})

	type groupParams struct {
		fx.In
		Options []health.CheckerOption `group:"health_check_options"`
	}

	app := fx.New(
		fx.Supply(&cfg, zap.NewNop().Sugar()),
		fx.Provide(newHealthCheckFx),
		fx.Invoke(func(p groupParams) {
			require.Len(t, p.Options, 1)

			checker := health.NewChecker(p.Options...)
			defer checker.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			res := checker.Check(ctx)

			_, ok := res.Details[backendElasticsearch]
			require.True(t, ok)
			require.Eventually(t, func() bool {
				res := checker.Check(ctx)
				es, ok := res.Details[backendElasticsearch]

				return ok && es.Status == health.StatusInactive
			}, 2*time.Second, 10*time.Millisecond)
		}),
	)
	require.NoError(t, app.Err())
}

func TestHealthCheck_ActiveTracksRuntimeBackend(t *testing.T) {
	t.Parallel()

	cfg := concurrency.AtomicValue[indexer.SearchConfig]{}
	cfg.Set(indexer.SearchConfig{Backend: backendPostgresql})

	check := healthCheck(&cfg)
	require.NotNil(t, check.IsActive)
	assert.False(t, check.IsActive())

	cfg.Set(indexer.SearchConfig{Backend: backendElasticsearch})
	assert.True(t, check.IsActive())

	cfg.Set(indexer.SearchConfig{Backend: backendPostgresql})
	assert.False(t, check.IsActive())
}

func TestHealthCheck_PingsCurrentAddresses(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"test"}`))
	}))
	defer srv.Close()

	cfg := concurrency.AtomicValue[indexer.SearchConfig]{}
	cfg.Set(indexer.SearchConfig{
		Backend: backendElasticsearch,
		Elasticsearch: elasticsearch.Config{
			Addresses: []string{srv.URL},
		},
	})

	check := healthCheck(&cfg)
	assert.True(t, check.IsActive())

	err := check.Check(context.Background())
	assert.NoError(t, err)
}

func TestHealthCheck_UnreachableWhenActive(t *testing.T) {
	t.Parallel()

	cfg := concurrency.AtomicValue[indexer.SearchConfig]{}
	cfg.Set(indexer.SearchConfig{
		Backend: backendElasticsearch,
		Elasticsearch: elasticsearch.Config{
			Addresses: []string{"http://127.0.0.1:1"},
		},
	})

	check := healthCheck(&cfg)
	err := check.Check(context.Background())
	assert.Error(t, err)
}
