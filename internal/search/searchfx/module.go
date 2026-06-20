package searchfx

import (
	"github.com/hexsans/hexmagnet/internal/database/db"
	dbsearch "github.com/hexsans/hexmagnet/internal/database/search"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	esearch "github.com/hexsans/hexmagnet/internal/elasticsearch/search"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	SearchCfg indexer.SearchConfig

	Queries utils.Lazy[*db.Queries]
	Logger  *zap.SugaredLogger
}

type Result struct {
	fx.Out

	Search        utils.Lazy[search.Search]
	SearchRuntime *search.Runtime
}

func newSearch(cfg indexer.SearchConfig, queries *db.Queries, logger *zap.SugaredLogger) search.Search {
	pg := dbsearch.NewFromPool(queries)

	if cfg.Backend != "elasticsearch" {
		return pg
	}

	esClient, err := elasticsearch.NewClient(cfg.Elasticsearch)
	if err != nil {
		logger.Warnw("failed to create ES client, falling back to PG only", "error", err)
		return pg
	}

	es := esearch.New(
		esClient,
		queries,
		embedding.NewCachingClient(embedding.NewClient(cfg.Elasticsearch.Embedding)),
		logger,
		cfg.Elasticsearch.Embedding.InstructionEnabled,
	)

	return search.NewRouter(es, pg)
}

func New(p Params) Result {
	runtime := search.NewRuntime(p.SearchCfg)

	runtime.SetSearchFactory(func(cfg indexer.SearchConfig) search.Search {
		q, err := p.Queries.Get()
		if err != nil {
			return nil
		}

		return newSearch(cfg, q, p.Logger)
	})

	return Result{
		Search: utils.NewLazy(func() (search.Search, error) {
			q, err := p.Queries.Get()
			if err != nil {
				return nil, err
			}

			s := newSearch(p.SearchCfg, q, p.Logger)
			runtime.Search.Set(s)

			return s, nil
		}),
		SearchRuntime: runtime,
	}
}
