package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SearchConfig struct {
	Backend       string               `validate:"oneof=postgresql elasticsearch" yaml:"backend"`
	Elasticsearch elasticsearch.Config `                                          yaml:"elasticsearch"`
}

const backendElasticsearch = "elasticsearch"

func DefaultSearchConfig() SearchConfig {
	return SearchConfig{Backend: "postgresql", Elasticsearch: elasticsearch.DefaultConfig()}
}

type Params struct {
	fx.In
	SearchCfg        SearchConfig
	ESClient         *elasticsearch.Client
	SearchConfigAtom *concurrency.AtomicValue[SearchConfig]
	Queries          utils.Lazy[*db.Queries]
	ConsumerMaker    queue.ConsumerMaker
	Logger           *zap.SugaredLogger
	Runtime          *queue.Runtime
	ConfigNotifier   *ConfigNotifier
	ReindexTracker   *ReindexTracker
}

type Result struct {
	fx.Out
	Worker worker.Worker `group:"workers"`
}

func New(p Params) Result {
	logger := p.Logger.Named("message_queue.enrich.indexer")

	var wg sync.WaitGroup

	return Result{
		Worker: worker.NewWorker("message_queue_enrich_indexer", fx.Hook{
			OnStart: func(ctx context.Context) error {
				q, err := p.Queries.Get()
				if err != nil {
					return err
				}

				startup := p.SearchConfigAtom.Get()

				var (
					esClient   *elasticsearch.Client
					lastESAddr []string
				)

				if startup.Backend == backendElasticsearch {
					esClient = p.ESClient
					lastESAddr = startup.Elasticsearch.Addresses
					ensureIndex(ctx, p.ESClient, dimsOrDefault(startup.Elasticsearch.Embedding.Dimensions), logger)
				}

				wg.Add(1)
				go runManagedIndexer(ctx, managedIndexerParams{
					cm:               p.ConsumerMaker,
					runtime:          p.Runtime,
					logger:           logger,
					searchConfigAtom: p.SearchConfigAtom,
					configNotifier:   p.ConfigNotifier,
					reindexTracker:   p.ReindexTracker,
					queries:          q,
					startES:          esClient,
					lastESAddr:       lastESAddr,
					wg:               &wg,
				})

				return nil
			},
			OnStop: func(_ context.Context) error {
				wg.Wait()
				return nil
			},
		}),
	}
}

func dimsOrDefault(dims int) int {
	if dims <= 0 {
		return 1024
	}

	return dims
}

func getEmbedder(searchConfigAtom *concurrency.AtomicValue[SearchConfig]) *embedding.Client {
	cfg := searchConfigAtom.Get()
	if cfg.Backend == backendElasticsearch {
		return embedding.NewClient(cfg.Elasticsearch.Embedding)
	}

	return nil
}

type managedIndexerParams struct {
	cm               queue.ConsumerMaker
	runtime          *queue.Runtime
	logger           *zap.SugaredLogger
	searchConfigAtom *concurrency.AtomicValue[SearchConfig]
	configNotifier   *ConfigNotifier
	reindexTracker   *ReindexTracker
	queries          *db.Queries
	startES          *elasticsearch.Client
	lastESAddr       []string
	wg               *sync.WaitGroup
}

func runManagedIndexer(ctx context.Context, p managedIndexerParams) {
	cm := p.cm
	runtime := p.runtime
	logger := p.logger
	searchConfigAtom := p.searchConfigAtom
	configNotifier := p.configNotifier
	reindexTracker := p.reindexTracker
	queries := p.queries
	wg := p.wg
	lastESAddr := p.lastESAddr

	defer wg.Done()

	backendCh, unsubscribe := runtime.SubscribeBackendChanges()
	defer unsubscribe()

	var (
		esClient atomic.Pointer[elasticsearch.Client]
		embedder atomic.Pointer[embedding.Client]
	)

	esClient.Store(p.startES)
	embedder.Store(getEmbedder(searchConfigAtom))

	var lastEmbeddingCfg embedding.Config
	if cfg := searchConfigAtom.Get(); cfg.Backend == backendElasticsearch {
		lastEmbeddingCfg = cfg.Elasticsearch.Embedding
	}

	embeddingCh, unsubscribeEmbedding := configNotifier.Subscribe()
	defer unsubscribeEmbedding()

	handler := func(ctx context.Context, _ string, value []byte) error {
		cfg := searchConfigAtom.Get()

		es := esClient.Load()
		if cfg.Backend != backendElasticsearch || es == nil {
			return nil
		}

		var ids []string
		if err := json.Unmarshal(value, &ids); err != nil {
			logger.Errorw("failed to unmarshal enriched IDs", "error", err)
			return err
		}

		if len(ids) == 0 {
			return nil
		}

		docs := make([]TorrentContentDocument, 0, len(ids))

		for _, id := range ids {
			infoHash, _, _, _, parseErr := parseCompositeID(id)
			if parseErr != nil {
				logger.Errorw("failed to parse composite ID", "id", id, "error", parseErr)
				continue
			}

			t, fetchErr := loadTorrent(ctx, queries, infoHash)
			if fetchErr != nil {
				logger.Errorw("failed to fetch torrent", "info_hash", infoHash, "error", fetchErr)
				continue
			}

			docs = append(docs, NewDocument(*t))
		}

		if len(docs) == 0 {
			return nil
		}

		if currentEmbedder := embedder.Load(); currentEmbedder != nil {
			texts := make([]string, len(docs))
			for i, doc := range docs {
				texts[i] = BuildSearchText(doc)
			}

			vectors, embedErr := currentEmbedder.Embed(ctx, texts)
			if embedErr != nil {
				logger.Warnw("failed to generate embeddings, continuing without vectors", "error", embedErr)
			} else {
				for i := range docs {
					docs[i].SearchVector = vectors[i]
				}
			}
		}

		body, bulkErr := BulkBody(docs)
		if bulkErr != nil {
			logger.Errorw("failed to build bulk body", "error", bulkErr)
			return bulkErr
		}

		if indexErr := es.BulkIndex(ctx, bytes.NewReader(body)); indexErr != nil {
			logger.Warnw("failed to bulk index documents, skipping batch", "error", indexErr)
			return nil
		}

		logger.Debugw("indexed torrent contents to ES", "count", len(docs))

		return nil
	}

	for {
		consumer, err := cm.NewConsumer(
			kafka.TopicEnriched,
			"hexmagnet-indexer",
			handler,
			logger,
		)
		if err != nil {
			logger.Errorw("failed to create consumer", "error", err)

			select {
			case <-time.After(time.Second):
				continue
			case <-ctx.Done():
				return
			}
		}

		if err := consumer.Start(ctx); err != nil {
			logger.Errorw("failed to start consumer", "error", err)
			continue
		}

		logger.Infow("consumer started", "topic", kafka.TopicEnriched)

		select {
		case <-backendCh:
			logger.Infow("backend changed, restarting consumer")
			embedder.Store(getEmbedder(searchConfigAtom))

			if cfg := searchConfigAtom.Get(); cfg.Backend == backendElasticsearch {
				lastEmbeddingCfg = cfg.Elasticsearch.Embedding
			}

			if err := consumer.Stop(ctx); err != nil {
				logger.Errorw("failed to stop consumer", "error", err)
			}
		case <-embeddingCh:
			currentCfg := searchConfigAtom.Get()
			if currentCfg.Backend != backendElasticsearch {
				esClient.Store(nil)
				embedder.Store(nil)

				lastESAddr = nil
				lastEmbeddingCfg = embedding.Config{}

				continue
			}

			es := esClient.Load()
			if es == nil || !reflect.DeepEqual(currentCfg.Elasticsearch.Addresses, lastESAddr) {
				newClient, clientErr := elasticsearch.NewClient(currentCfg.Elasticsearch)
				if clientErr != nil {
					logger.Errorw("failed to create elasticsearch client", "error", clientErr)
					continue
				}

				esClient.Store(newClient)
				es = newClient
				lastESAddr = currentCfg.Elasticsearch.Addresses
				ensureIndex(ctx, es, dimsOrDefault(currentCfg.Elasticsearch.Embedding.Dimensions), logger)
			}

			if reflect.DeepEqual(currentCfg.Elasticsearch.Embedding, lastEmbeddingCfg) {
				continue
			}

			oldDims := lastEmbeddingCfg.Dimensions
			lastEmbeddingCfg = currentCfg.Elasticsearch.Embedding

			embedder.Store(getEmbedder(searchConfigAtom))

			if oldDims != 0 && currentCfg.Elasticsearch.Embedding.Dimensions != oldDims {
				logger.Infow("embedding dimensions changed, recreating index and reindexing",
					"old", oldDims, "new", currentCfg.Elasticsearch.Embedding.Dimensions)

				if err := reindexTracker.Start(
					context.WithoutCancel(ctx),
					es,
					embedder.Load(),
					queries,
					currentCfg.Elasticsearch.Embedding.Dimensions,
					logger,
				); err != nil {
					logger.Warnw("auto-reindex skipped", "error", err)
				}
			} else {
				logger.Infow("embedding config changed, recreated embedder")
			}
		case <-ctx.Done():
			if err := consumer.Stop(ctx); err != nil {
				logger.Errorw("failed to stop consumer on shutdown", "error", err)
			}

			return
		}
	}
}

func ensureIndex(ctx context.Context, es *elasticsearch.Client, dims int, logger *zap.SugaredLogger) {
	exists, err := es.IndexExists(ctx, IndexName)
	if err != nil {
		logger.Warnw("failed to check index existence, attempting create", "index", IndexName, "error", err)
	} else if exists {
		logger.Debugw("index already exists", "index", IndexName)
		return
	}

	if err := es.CreateIndex(ctx, IndexName, bytes.NewReader([]byte(IndexMapping(dims)))); err != nil {
		logger.Infow("index may already exist, continuing", "index", IndexName, "error", err)
	} else {
		logger.Infow("created elasticsearch index", "index", IndexName)
	}
}
