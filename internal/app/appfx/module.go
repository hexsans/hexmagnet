package appfx

import (
	"context"

	"github.com/hexsans/hexmagnet/internal/blocking"
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/database"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/dht/pipelinefx"
	dhtcrawlerPkg "github.com/hexsans/hexmagnet/internal/dhtcrawler"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/elasticsearchfx"
	"github.com/hexsans/hexmagnet/internal/gql/gqlfx"
	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/httpserver"
	"github.com/hexsans/hexmagnet/internal/jobcontrol"
	"github.com/hexsans/hexmagnet/internal/logging"
	"github.com/hexsans/hexmagnet/internal/metrics/torrentmetrics"
	"github.com/hexsans/hexmagnet/internal/processor"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	dhtPkg "github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/dhtfx"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/responder"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/server"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainfofx"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/retryqueue"
	search "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/search/searchfx"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"github.com/hexsans/hexmagnet/internal/torrent"
	"github.com/hexsans/hexmagnet/internal/torrentstore"
	"github.com/hexsans/hexmagnet/internal/torznab"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/hexsans/hexmagnet/internal/version"
	"github.com/hexsans/hexmagnet/internal/webhook"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func New(queueCfg queue.Config) fx.Option {
	return fx.Module(
		"app",
		blocking.NewModule(),
		classifier.NewModule(),
		dhtcrawlerPkg.NewModule(),
		dhtfx.New(),
		pipelinefx.New(),
		database.NewModule(),
		servercfg.NewModule(),
		elasticsearchfx.New(),
		torrentstore.NewModule(),
		fx.Provide(torznab.New),
		fx.Provide(webhook.New),
		fx.Invoke(func(
			handler *torznab.Handler,
			cm *configmgr.Manager,
		) {
			if cm == nil {
				return
			}

			cm.Subscribe("torznab",
				func(_ context.Context, snap *configmgr.Snapshot) error {
					handler.Update(snap.Torznab)
					return nil
				}, configmgr.ApplyAsync)
		}),
		fx.Invoke(func(
			publisher *webhook.Publisher,
			cm *configmgr.Manager,
		) {
			if cm == nil {
				return
			}

			cm.Subscribe("webhooks",
				func(_ context.Context, snap *configmgr.Snapshot) error {
					publisher.Update(snap.Webhooks)
					return nil
				}, configmgr.ApplyAsync)
		}),
		fx.Invoke(func(
			retryQueue *retryqueue.Queue,
			cm *configmgr.Manager,
		) {
			if cm == nil {
				return
			}

			cm.Subscribe("retry_queue",
				func(_ context.Context, snap *configmgr.Snapshot) error {
					retryQueue.UpdateConfig(snap.RetryQueue)
					return nil
				}, configmgr.ApplyAsync)
		}),
		fx.Provide(
			func(cfg dhtPkg.Config) pipelinefx.Config {
				return pipelinefx.Config{
					Port:                         cfg.Port,
					Responder:                    pipelinefx.ResponderConfig(cfg.Responder),
					BootstrapNodes:               cfg.BootstrapNodes,
					ReseedBootstrapNodesInterval: cfg.ReseedBootstrapNodesInterval,
				}
			},
			func(cfg indexer.SearchConfig) elasticsearch.Config {
				return cfg.Elasticsearch
			},
			func(c pipelinefx.Config) server.Config {
				return server.Config{
					Port:             c.Port,
					ResponderEnabled: c.Responder.Enabled,
				}
			},
			func(c pipelinefx.Config) responder.Config {
				return responder.Config{
					GlobalRateLimit: c.Responder.GlobalRateLimit,
					PerIPRateLimit:  c.Responder.PerIPRateLimit,
				}
			},
			func(c pipelinefx.Config, req metainforequester.Config, srv servercfg.Config) dhtcrawlerPkg.Config {
				return dhtcrawlerPkg.Config{
					BootstrapNodes:               c.BootstrapNodes,
					ReseedBootstrapNodesInterval: c.ReseedBootstrapNodesInterval,
					RescrapeThreshold:            req.RescrapeThreshold,
					HashDiscoverLimit:            req.HashDiscoverLimit,
					EmbedTrackers:                srv.EmbedTrackers,
				}
			},
			fx.Annotated{
				Name: "global_request_limiter",
				Target: func(req metainforequester.Config) *rate.Limiter {
					if req.RequestLimit <= 0 {
						return nil
					}

					return rate.NewLimiter(rate.Limit(req.RequestLimit), req.RequestLimit)
				},
			},
			indexer.NewConfigNotifier,
			indexer.NewReindexTracker,
			jobcontrol.NewController,
			jobcontrol.NewStateStore,
			func(c *jobcontrol.Controller) dhtcrawlerPkg.PauseGate {
				return c
			},
		),
		fx.Invoke(func(
			lc fx.Lifecycle,
			reindexTracker *indexer.ReindexTracker,
			reclassifyTracker *processor.ReclassifyTracker,
			searchCfg indexer.SearchConfig,
			classifierCfg classifier.Config,
			logger *zap.SugaredLogger,
		) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					if err := reindexTracker.Load(ctx, indexer.Fingerprint(searchCfg), logger); err != nil {
						logger.Warnw("failed to restore reindex progress", "error", err)
					}

					if err := reclassifyTracker.Load(ctx, processor.Fingerprint(classifierCfg), logger); err != nil {
						logger.Warnw("failed to restore reclassify progress", "error", err)
					}

					return nil
				},
			})
		}),
		fx.Provide(newConfigManager),
		fx.Invoke(func(cm *configmgr.Manager, logger *zap.SugaredLogger) {
			cm.SetLogger(logger)
		}),
		fx.Provide(searchfx.New),
		fx.Provide(func(r *search.Runtime) *concurrency.AtomicValue[indexer.SearchConfig] {
			return r.SearchConfig
		}),
		fx.Invoke(func(s utils.Lazy[search.Search]) {
			_, _ = s.Get()
		}),
		gqlfx.New(),
		health.NewModule(),
		httpserver.NewModule(),
		logging.NewModule(),
		metainfofx.New(),
		fx.Provide(torrentmetrics.New),
		processor.NewProcessorFxModule(),
		queue.NewModule(queueCfg),
		tmdb.NewModule(),
		torrent.NewModule(),

		version.NewModule(),
		worker.NewModule(),
		fx.Provide(httpserver.NewWebUI),
	)
}

//nolint:revive // config bundle injected by the fx container
func newConfigManager(
	lc fx.Lifecycle,
	serverCfg servercfg.Config,
	dhtCfg dhtPkg.Config,
	classifierCfg classifier.Config,
	postgresCfg postgres.Config,
	searchCfg indexer.SearchConfig,
	queueCfg queue.Config,
	dhtRequesterCfg metainforequester.Config,
	torznabCfg torznab.Config,
	webhooksCfg webhook.Config,
	retryQueueCfg retryqueue.Config,
) *configmgr.Manager {
	initial := &configmgr.Snapshot{
		DHTRequester: dhtRequesterCfg,
		DHT:          dhtCfg,
		Server:       serverCfg,
		Classifier:   classifierCfg,
		Postgres:     postgresCfg,
		Queue:        queueCfg,
		Search:       searchCfg,
		Torznab:      torznabCfg,
		Webhooks:     webhooksCfg,
		RetryQueue:   retryQueueCfg,
	}

	mgr := configmgr.NewManager(initial, nil)

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			mgr.Stop()
			return nil
		},
	})

	return mgr
}
