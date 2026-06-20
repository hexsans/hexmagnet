package elasticsearchfx

import (
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	backendElasticsearch = "elasticsearch"
	backendPostgresql    = "postgresql"
)

type newClientParams struct {
	fx.In
	Config    elasticsearch.Config
	SearchCfg indexer.SearchConfig
	Logger    *zap.SugaredLogger
}

func newClientIfEnabled(p newClientParams) (*elasticsearch.Client, error) {
	if p.SearchCfg.Backend != backendElasticsearch {
		p.Logger.Infow("elasticsearch not enabled, search backend is not 'elasticsearch'")
		return nil, nil //nolint:nilnil // optional provider: nil client means backend disabled
	}

	return elasticsearch.NewClient(p.Config)
}

func newEmbedderIfEnabled(cfg elasticsearch.Config) *embedding.Client {
	return embedding.NewClient(cfg.Embedding)
}

func New() fx.Option {
	return fx.Module(
		backendElasticsearch,
		fx.Provide(
			newClientIfEnabled,
			newEmbedderIfEnabled,
			newHealthCheckFx,
		),
	)
}
