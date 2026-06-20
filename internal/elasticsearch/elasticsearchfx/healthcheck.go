package elasticsearchfx

import (
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"go.uber.org/fx"
)

type healthCheckFxParams struct {
	fx.In
	SearchConfig *concurrency.AtomicValue[indexer.SearchConfig]
}

type healthCheckFxResult struct {
	fx.Out
	Option health.CheckerOption `group:"health_check_options"`
}

func healthCheck(searchConfig *concurrency.AtomicValue[indexer.SearchConfig]) health.Check {
	return elasticsearch.NewHealthCheck(
		func() bool {
			return searchConfig.Get().Backend == backendElasticsearch
		},
		func() (*elasticsearch.Client, error) {
			return elasticsearch.NewClient(searchConfig.Get().Elasticsearch)
		},
	)
}

func newHealthCheckFx(p healthCheckFxParams) healthCheckFxResult {
	return healthCheckFxResult{
		Option: health.WithPeriodicCheck(
			time.Second*30,
			0,
			healthCheck(p.SearchConfig),
		),
	}
}
