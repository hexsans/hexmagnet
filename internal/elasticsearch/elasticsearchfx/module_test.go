package elasticsearchfx

import (
	"testing"

	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	t.Parallel()

	opt := New()
	assert.NotNil(t, opt)

	app := fxtest.New(t,
		fx.Supply(
			elasticsearch.DefaultConfig(),
			indexer.DefaultSearchConfig(),
			zap.NewNop().Sugar(),
		),
		opt,
	)
	defer app.RequireStart().RequireStop()
}

func TestNewClientIfEnabled_Elasticsearch(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop().Sugar()
	p := newClientParams{
		Config:    elasticsearch.Config{Addresses: []string{"http://localhost:9200"}},
		SearchCfg: indexer.SearchConfig{Backend: backendElasticsearch},
		Logger:    logger,
	}

	client, err := newClientIfEnabled(p)
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestNewClientIfEnabled_PostgreSQL(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop().Sugar()
	p := newClientParams{
		Config:    elasticsearch.DefaultConfig(),
		SearchCfg: indexer.SearchConfig{Backend: backendPostgresql},
		Logger:    logger,
	}

	client, err := newClientIfEnabled(p)
	require.NoError(t, err)
	assert.Nil(t, client)
}

func TestNewClientIfEnabled_InvalidAddress(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop().Sugar()
	p := newClientParams{
		Config:    elasticsearch.Config{Addresses: []string{"://invalid"}},
		SearchCfg: indexer.SearchConfig{Backend: backendElasticsearch},
		Logger:    logger,
	}

	client, err := newClientIfEnabled(p)
	require.Error(t, err)
	assert.Nil(t, client)
}

func TestNew_FxIntegration(t *testing.T) {
	t.Parallel()

	var esClient *elasticsearch.Client

	app := fxtest.New(t,
		fx.Supply(
			elasticsearch.DefaultConfig(),
			indexer.DefaultSearchConfig(),
			zap.NewNop().Sugar(),
		),
		New(),
		fx.Populate(&esClient),
	)
	defer app.RequireStart().RequireStop()

	assert.Nil(t, esClient)
}
