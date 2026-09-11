package resolvers

import (
	"sync/atomic"

	"github.com/hexsans/hexmagnet/internal/blocking"
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/dhtcrawler"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/jobcontrol"
	"github.com/hexsans/hexmagnet/internal/logging"
	"github.com/hexsans/hexmagnet/internal/metrics/torrentmetrics"
	"github.com/hexsans/hexmagnet/internal/processor"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/retryqueue"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"github.com/hexsans/hexmagnet/internal/torznab"
	"github.com/hexsans/hexmagnet/internal/webhook"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/zap"
)

type Resolver struct {
	DB                   *db.Queries
	Workers              worker.Registry
	Checker              health.Checker
	TorrentMetricsClient torrentmetrics.Client
	Processor            processor.Processor
	BlockingManager      blocking.Manager
	Producer             queue.Producer
	Logger               *zap.SugaredLogger
	RetryQueueSvc        *retryqueue.Queue

	ServerCfg       servercfg.Config
	DhtCfg          dht.Config
	ClassifierCfg   classifier.Config
	TmdbCfg         tmdb.Config
	PostgresCfg     postgres.Config
	SearchCfg       indexer.SearchConfig
	QueueCfg        queue.Config
	DHTRequesterCfg metainforequester.Config
	TorznabCfg      torznab.Config
	WebhooksCfg     webhook.Config
	RetryQueueCfg   retryqueue.Config

	ConfigManager     *configmgr.Manager
	DhtCrawlerRuntime *dhtcrawler.Runtime
	LogManager        *logging.Manager
	ConfigFilePath    string
	PostgresRuntime   *postgres.Runtime
	QueueRuntime      *queue.Runtime
	SearchRuntime     *dbsearch.Runtime
	ReindexTracker    *indexer.ReindexTracker
	ReclassifyTracker *processor.ReclassifyTracker
	JobControl        *jobcontrol.Controller

	// ConfigValidator runs pre-flight checks on config updates. When nil the
	// default validator with live probes is used. Tests override it.
	ConfigValidator *ConfigValidator

	esClientPtr atomic.Pointer[elasticsearch.Client]
	embedderPtr atomic.Pointer[embedding.Client]
}

func (r *Resolver) ES() *elasticsearch.Client     { return r.esClientPtr.Load() }
func (r *Resolver) EM() *embedding.Client         { return r.embedderPtr.Load() }
func (r *Resolver) SetES(c *elasticsearch.Client) { r.esClientPtr.Store(c) }
func (r *Resolver) SetEM(e *embedding.Client)     { r.embedderPtr.Store(e) }
