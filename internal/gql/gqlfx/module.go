package gqlfx

import (
	"context"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/hexsans/hexmagnet/internal/blocking"
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/dhtcrawler"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/gql"
	"github.com/hexsans/hexmagnet/internal/gql/httpserver"
	"github.com/hexsans/hexmagnet/internal/gql/resolvers"
	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/logging"
	"github.com/hexsans/hexmagnet/internal/metrics/torrentmetrics"
	"github.com/hexsans/hexmagnet/internal/processor"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/responder"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/hexsans/hexmagnet/internal/queue"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/hexsans/hexmagnet/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func New() fx.Option {
	return fx.Module(
		"graphql",
		fx.Provide(
			NewGqlConfig,
			httpserver.New,
			func(
				lcfg utils.Lazy[gql.Config],
			) utils.Lazy[graphql.ExecutableSchema] {
				return utils.NewLazy(func() (graphql.ExecutableSchema, error) {
					cfg, err := lcfg.Get()
					if err != nil {
						return nil, err
					}

					return gql.NewExecutableSchema(cfg), nil
				})
			},
		),
		fx.Provide(
			func(p Params) Result {
				return Result{
					Resolver: utils.NewLazy(func() (*resolvers.Resolver, error) {
						ch, err := p.Checker.Get()
						if err != nil {
							return nil, err
						}

						tm, err := p.TorrentMetricsClient.Get()
						if err != nil {
							return nil, err
						}

						pr, err := p.Processor.Get()
						if err != nil {
							return nil, err
						}

						bm, err := p.BlockingManager.Get()
						if err != nil {
							return nil, err
						}

						q, err := p.Queries.Get()
						if err != nil {
							return nil, err
						}

						r := &resolvers.Resolver{
							DB:                   q,
							Checker:              ch,
							TorrentMetricsClient: tm,
							Processor:            pr,
							BlockingManager:      bm,
							Logger:               p.Logger,
							ServerCfg:            p.ServerCfg,
							DhtCfg:               p.DhtCfg,
							ClassifierCfg:        p.ClassifierCfg,
							TmdbCfg:              p.TmdbCfg,
							PostgresCfg:          p.PostgresCfg,
							SearchCfg:            p.SearchCfg,
							QueueCfg:             p.QueueCfg,
							DHTRequesterCfg:      p.DHTRequesterCfg,
							ConfigManager:        p.ConfigManager,
							DhtCrawlerRuntime:    p.DhtCrawlerRuntime,
							LogManager:           p.LogManager,
							ConfigFilePath:       "./hexmagnet.yaml",
							PostgresRuntime:      p.PostgresRuntime,
							QueueRuntime:         p.QueueRuntime,
							SearchRuntime:        p.SearchRuntime,
							ReindexTracker:       p.ReindexTracker,
						}
						r.SetES(p.ESClient)
						r.SetEM(p.Embedder)

						return r, nil
					}),
				}
			},
		),
		fx.Invoke(func(
			resolver utils.Lazy[*resolvers.Resolver],
			workers worker.Registry,
		) {
			resolver.Decorate(func(r *resolvers.Resolver) (*resolvers.Resolver, error) {
				r.Workers = workers
				return r, nil
			})
		}),
		fx.Invoke(registerGlobalSubscribers),
	)
}

type GlobalSubscriberParams struct {
	fx.In
	ConfigManager        *configmgr.Manager
	RequestLimiter       *rate.Limiter     `name:"global_request_limiter" optional:"true"`
	ResponderEnabled     *atomic.Bool      `                              optional:"true"`
	ResponderRateLimiter responder.Limiter `                              optional:"true"`
	PostgresRuntime      *postgres.Runtime
	QueueRuntime         *queue.Runtime
	SearchRuntime        *dbsearch.Runtime
	Resolve              utils.Lazy[*resolvers.Resolver]
	Queries              utils.Lazy[*db.Queries]
	Logger               *zap.SugaredLogger

	DHTRequesterCfg metainforequester.Config
	DhtCfg          dht.Config
	QueueCfg        queue.Config
	SearchCfg       indexer.SearchConfig

	ConfigNotifier *indexer.ConfigNotifier
	TmdbUpdater    tmdb.ConfigUpdater `optional:"true"`
}

func registerGlobalSubscribers(p GlobalSubscriberParams) {
	if p.ConfigManager == nil {
		return
	}

	if p.RequestLimiter != nil {
		last := p.DHTRequesterCfg.RequestLimit
		p.ConfigManager.Subscribe(context.Background(), "request_limiter",
			func(_ context.Context, snap *configmgr.Snapshot) error {
				if snap.DHTRequester.RequestLimit == last {
					return nil
				}

				last = snap.DHTRequester.RequestLimit
				if last > 0 {
					p.RequestLimiter.SetLimit(rate.Limit(last))
				}

				return nil
			}, configmgr.ApplyAsync)
	}

	if p.ResponderEnabled != nil {
		last := p.DhtCfg.Responder.Enabled
		p.ConfigManager.Subscribe(context.Background(), "responder_enabled",
			func(_ context.Context, snap *configmgr.Snapshot) error {
				if snap.DHT.Responder.Enabled == last {
					return nil
				}

				last = snap.DHT.Responder.Enabled
				p.ResponderEnabled.Store(last)

				return nil
			}, configmgr.ApplyAsync)
	}

	if p.ResponderRateLimiter != nil {
		lastGlobal := p.DhtCfg.Responder.GlobalRateLimit
		lastPerIP := p.DhtCfg.Responder.PerIPRateLimit
		p.ConfigManager.Subscribe(context.Background(), "responder_rate_limiter",
			func(_ context.Context, snap *configmgr.Snapshot) error {
				if snap.DHT.Responder.GlobalRateLimit == lastGlobal &&
					snap.DHT.Responder.PerIPRateLimit == lastPerIP {
					return nil
				}

				lastGlobal = snap.DHT.Responder.GlobalRateLimit
				lastPerIP = snap.DHT.Responder.PerIPRateLimit

				p.ResponderRateLimiter.SetGlobalRateLimit(lastGlobal)
				p.ResponderRateLimiter.SetPerIPRateLimit(lastPerIP)

				return nil
			}, configmgr.ApplyAsync)
	}

	if p.TmdbUpdater != nil {
		var last tmdb.Config

		p.ConfigManager.Subscribe(context.Background(), "tmdb",
			func(_ context.Context, snap *configmgr.Snapshot) error {
				if snap.Classifier.Tmdb == last {
					return nil
				}

				last = snap.Classifier.Tmdb
				p.TmdbUpdater.UpdateConfig(last)

				return nil
			}, configmgr.ApplyAsync)
	}

	if p.PostgresRuntime != nil {
		p.PostgresRuntime.SetOnPoolReplaced(func(oldPool *pgxpool.Pool) {
			q, err := p.Queries.Get()
			if err != nil {
				return
			}

			q.ReplacePool(p.PostgresRuntime.GetPool())

			go func() {
				time.Sleep(30 * time.Second)

				if oldPool != nil {
					oldPool.Close()
				}
			}()
		})
	}

	if p.QueueRuntime != nil {
		last := p.QueueCfg
		p.ConfigManager.Subscribe(context.Background(), "queue",
			func(_ context.Context, snap *configmgr.Snapshot) error {
				if reflect.DeepEqual(snap.Queue, last) {
					return nil
				}

				last = snap.Queue

				return p.QueueRuntime.SwitchTo(snap.Queue) //nolint:contextcheck // SwitchTo accepts no context
			}, configmgr.ApplySync)
	}

	if p.SearchRuntime != nil {
		last := p.SearchCfg
		p.ConfigManager.Subscribe(context.Background(), "search",
			func(_ context.Context, snap *configmgr.Snapshot) error {
				if reflect.DeepEqual(snap.Search, last) {
					return nil
				}

				last = snap.Search
				p.SearchRuntime.SwitchBackend(snap.Search)

				if p.ConfigNotifier != nil {
					p.ConfigNotifier.Notify()
				}

				return nil
			}, configmgr.ApplySync)
	}

	if p.PostgresRuntime != nil {
		p.ConfigManager.Subscribe(context.Background(), "postgres",
			func(ctx context.Context, snap *configmgr.Snapshot) error {
				if p.PostgresRuntime.GetConfig() == snap.Postgres {
					return nil
				}

				oldPool, err := p.PostgresRuntime.Reconnect(ctx, snap.Postgres, p.Logger)
				if err != nil {
					return err
				}

				go func() {
					time.Sleep(30 * time.Second)

					if oldPool != nil {
						oldPool.Close()
					}
				}()

				return nil
			}, configmgr.ApplySync)
	}

	lastSearch := p.SearchCfg
	p.ConfigManager.Subscribe(context.Background(), "elasticsearch",
		func(_ context.Context, snap *configmgr.Snapshot) error {
			if reflect.DeepEqual(snap.Search, lastSearch) {
				return nil
			}

			lastSearch = snap.Search

			r, err := p.Resolve.Get()
			if err != nil {
				return err
			}

			if snap.Search.Backend == "elasticsearch" {
				esClient, esErr := elasticsearch.NewClient(snap.Search.Elasticsearch)
				if esErr != nil {
					return esErr
				}

				embedder := embedding.NewClient(snap.Search.Elasticsearch.Embedding)

				r.SetES(esClient)
				r.SetEM(embedder)
			} else {
				r.SetES(nil)
				r.SetEM(nil)
			}

			return nil
		}, configmgr.ApplySync)
}

type Params struct {
	fx.In
	Queries              utils.Lazy[*db.Queries]
	Checker              utils.Lazy[health.Checker]
	TorrentMetricsClient utils.Lazy[torrentmetrics.Client]
	Processor            utils.Lazy[processor.Processor]
	BlockingManager      utils.Lazy[blocking.Manager]
	Logger               *zap.SugaredLogger

	ServerCfg         servercfg.Config
	DhtCfg            dht.Config
	ClassifierCfg     classifier.Config
	TmdbCfg           tmdb.Config
	TmdbUpdater       tmdb.ConfigUpdater `optional:"true"`
	PostgresCfg       postgres.Config
	SearchCfg         indexer.SearchConfig
	QueueCfg          queue.Config
	DHTRequesterCfg   metainforequester.Config
	ConfigManager     *configmgr.Manager
	DhtCrawlerRuntime *dhtcrawler.Runtime `name:"dht_crawler_runtime" optional:"true"`
	LogManager        *logging.Manager    `optional:"true"`
	PostgresRuntime   *postgres.Runtime
	QueueRuntime      *queue.Runtime
	SearchRuntime     *dbsearch.Runtime
	ReindexTracker    *indexer.ReindexTracker

	ESClient *elasticsearch.Client
	Embedder *embedding.Client
}

type Result struct {
	fx.Out
	Resolver utils.Lazy[*resolvers.Resolver]
}
