package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sync/atomic"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/queue/permanent"
	"github.com/hexsans/hexmagnet/internal/queue/supervisor"
	"github.com/hexsans/hexmagnet/internal/retryqueue"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SearchConfig struct {
	Backend       string               `validate:"oneof=postgresql elasticsearch" yaml:"backend"`
	Elasticsearch elasticsearch.Config `                                          yaml:"elasticsearch"`
	// MaxSearchFiles bounds how many file paths from a single torrent are
	// included in the full-text search index (PG tsvector / ES embedding text).
	// Large torrents can contain thousands of files; capping keeps index size
	// bounded. <= 0 means no cap.
	MaxSearchFiles int `validate:"gte=0" yaml:"max_search_files"`
}

const backendElasticsearch = "elasticsearch"

func DefaultSearchConfig() SearchConfig {
	return SearchConfig{
		Backend:        "postgresql",
		Elasticsearch:  elasticsearch.DefaultConfig(),
		MaxSearchFiles: 30,
	}
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
	RetryQueue       *retryqueue.Queue `optional:"true"`
}

type Result struct {
	fx.Out
	Worker worker.Worker `group:"workers"`
}

func New(p Params) Result {
	logger := p.Logger.Named("message_queue.enrich.indexer")

	var (
		esClient atomic.Pointer[elasticsearch.Client]
		embedder atomic.Pointer[embedding.Client]

		queries          *db.Queries
		lastESAddr       []string
		lastEmbeddingCfg embedding.Config
	)

	sup := supervisor.New(supervisor.Params{
		Maker:   p.ConsumerMaker,
		Runtime: p.Runtime,
		Topic:   kafka.TopicEnriched,
		GroupID: "hexmagnet-indexer",
		Prepare: func(ctx context.Context) (queue.MessageHandler, error) {
			q, err := p.Queries.Get()
			if err != nil {
				return nil, err
			}

			queries = q

			startup := p.SearchConfigAtom.Get()

			if startup.Backend == backendElasticsearch {
				esClient.Store(p.ESClient)

				lastESAddr = startup.Elasticsearch.Addresses
				lastEmbeddingCfg = startup.Elasticsearch.Embedding

				ensureIndex(ctx, p.ESClient, dimsOrDefault(startup.Elasticsearch.Embedding.Dimensions), logger)
			}

			embedder.Store(getEmbedder(p.SearchConfigAtom))

			return func(ctx context.Context, _ string, value []byte) error {
				cfg := p.SearchConfigAtom.Get()

				es := esClient.Load()
				if cfg.Backend != backendElasticsearch || es == nil {
					return nil
				}

				var ids []string
				if err := json.Unmarshal(value, &ids); err != nil {
					logger.Errorw("failed to unmarshal enriched IDs", "error", err)

					return permanent.Mark(err)
				}

				if len(ids) == 0 {
					return nil
				}

				docs := make([]TorrentContentDocument, 0, len(ids))
				parsed := make([]parsedEnrichID, 0, len(ids))

				var (
					parseFailures int
					fetchFailures int
					fetchRetries  []parsedEnrichID
					lastFetchErr  error
				)

				for _, id := range ids {
					infoHash, _, _, _, parseErr := parseCompositeID(id)
					if parseErr != nil {
						parseFailures++

						continue
					}

					t, fetchErr := loadTorrent(ctx, queries, infoHash)
					if fetchErr != nil {
						fetchFailures++
						lastFetchErr = fetchErr

						fetchRetries = append(fetchRetries, parsedEnrichID{
							compositeID: id,
							infoHash:    infoHash.String(),
						})

						continue
					}

					parsed = append(parsed, parsedEnrichID{compositeID: id, infoHash: infoHash.String()})
					docs = append(docs, NewDocument(*t))
				}

				if len(fetchRetries) > 0 {
					enqueueEnrichRetries(ctx, p.RetryQueue, fetchRetries, lastFetchErr)
				}

				if parseFailures > 0 || fetchFailures > 0 {
					logger.Debugw("skipped enriched IDs",
						"parse_failures", parseFailures,
						"fetch_failures", fetchFailures,
						"total", len(ids),
					)
				}

				if len(docs) == 0 {
					return nil
				}

				if currentEmbedder := embedder.Load(); currentEmbedder != nil {
					texts := make([]string, len(docs))
					for i, doc := range docs {
						texts[i] = BuildSearchText(doc, cfg.MaxSearchFiles)
					}

					vectors, embedErr := currentEmbedder.Embed(ctx, texts)
					if embedErr != nil {
						logger.Debugw("failed to generate embeddings, continuing without vectors",
							"count", len(docs),
							"error", embedErr,
						)

						enqueueEnrichRetries(ctx, p.RetryQueue, parsed, embedErr)

						return nil
					}

					for i := range docs {
						docs[i].SearchVector = vectors[i]
					}
				}

				body, bulkErr := BulkBody(docs)
				if bulkErr != nil {
					logger.Errorw("failed to build bulk body", "error", bulkErr)

					enqueueEnrichRetries(ctx, p.RetryQueue, parsed, bulkErr)

					return nil
				}

				if indexErr := es.BulkIndex(ctx, bytes.NewReader(body)); indexErr != nil {
					logger.Debugw("failed to bulk index documents, skipping batch",
						"count", len(docs),
						"error", indexErr,
					)

					enqueueEnrichRetries(ctx, p.RetryQueue, parsed, indexErr)

					return nil
				}

				removeEnrichRetries(ctx, p.RetryQueue, parsed)

				logger.Debugw("indexed torrent contents to ES", "count", len(docs))

				return nil
			}, nil
		},
		Trigger: &supervisor.Trigger{
			Subscribe: p.ConfigNotifier.Subscribe,
			OnEvent: func(ctx context.Context) bool {
				currentCfg := p.SearchConfigAtom.Get()
				if currentCfg.Backend != backendElasticsearch {
					esClient.Store(nil)
					embedder.Store(nil)

					lastESAddr = nil
					lastEmbeddingCfg = embedding.Config{}

					return false
				}

				es := esClient.Load()
				if es == nil || !reflect.DeepEqual(currentCfg.Elasticsearch.Addresses, lastESAddr) {
					newClient, clientErr := elasticsearch.NewClient(currentCfg.Elasticsearch)
					if clientErr != nil {
						logger.Errorw("failed to create elasticsearch client", "error", clientErr)

						return false
					}

					esClient.Store(newClient)
					es = newClient
					lastESAddr = currentCfg.Elasticsearch.Addresses
					ensureIndex(ctx, es, dimsOrDefault(currentCfg.Elasticsearch.Embedding.Dimensions), logger)
				}

				if reflect.DeepEqual(currentCfg.Elasticsearch.Embedding, lastEmbeddingCfg) {
					return false
				}

				oldDims := lastEmbeddingCfg.Dimensions
				lastEmbeddingCfg = currentCfg.Elasticsearch.Embedding

				embedder.Store(getEmbedder(p.SearchConfigAtom))

				if oldDims != 0 && currentCfg.Elasticsearch.Embedding.Dimensions != oldDims {
					logger.Infow("embedding dimensions changed, recreating index and reindexing",
						"old", oldDims, "new", currentCfg.Elasticsearch.Embedding.Dimensions)

					if err := p.ReindexTracker.Start(
						context.WithoutCancel(ctx),
						es,
						embedder.Load(),
						queries,
						currentCfg.Elasticsearch.Embedding.Dimensions,
						currentCfg.MaxSearchFiles,
						Fingerprint(currentCfg),
						true,
						logger,
					); err != nil {
						logger.Warnw("auto-reindex skipped", "error", err)
					}
				} else {
					logger.Infow("embedding config changed, recreated embedder")
				}

				return false
			},
		},
		Logger: logger,
	})

	return Result{
		Worker: worker.NewWorker("message_queue_enrich_indexer", sup.Hook()),
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

// parsedEnrichID pairs a composite enriched ID with its info hash.
type parsedEnrichID struct {
	compositeID string
	infoHash    string
}

// enqueueEnrichRetries records indexing failures in the retry queue as a
// single batched upsert. When the retry queue is nil or disabled failures keep
// the legacy behaviour (logged and dropped).
func enqueueEnrichRetries(
	ctx context.Context,
	retryQueue *retryqueue.Queue,
	parsed []parsedEnrichID,
	cause error,
) {
	if retryQueue == nil {
		return
	}

	items := make([]retryqueue.RetryItem, 0, len(parsed))
	for _, id := range parsed {
		items = append(items, retryqueue.RetryItem{
			Stage:    retryqueue.StageEnrich,
			InfoHash: id.infoHash,
			Payload:  []string{id.compositeID},
		})
	}

	// EnqueueBatch logs failures internally.
	_ = retryQueue.EnqueueBatch(ctx, items, cause)
}

// removeEnrichRetries clears retry entries after a successful index run.
func removeEnrichRetries(
	ctx context.Context,
	retryQueue *retryqueue.Queue,
	parsed []parsedEnrichID,
) {
	if retryQueue == nil {
		return
	}

	hashes := make([]string, 0, len(parsed))
	for _, id := range parsed {
		hashes = append(hashes, id.infoHash)
	}

	retryQueue.Remove(ctx, hashes...)
}
