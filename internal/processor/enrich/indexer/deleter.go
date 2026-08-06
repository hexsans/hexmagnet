package indexer

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type DeleterParams struct {
	fx.In

	ESClient         *elasticsearch.Client
	SearchConfigAtom *concurrency.AtomicValue[SearchConfig]
	ConsumerMaker    queue.ConsumerMaker
	Logger           *zap.SugaredLogger
	Runtime          *queue.Runtime
	ConfigNotifier   *ConfigNotifier
}

func NewDeleter(p DeleterParams) Result {
	logger := p.Logger.Named("message_queue.enrich.deleter")

	var wg sync.WaitGroup

	esRef := &concurrency.AtomicValue[*elasticsearch.Client]{}
	esRef.Set(p.ESClient)

	return Result{
		Worker: worker.NewWorker("message_queue_enrich_deleter", fx.Hook{
			OnStart: func(ctx context.Context) error {
				wg.Add(1)
				go runDeleter(ctx, p.ConsumerMaker, p.Runtime, logger, esRef, p.SearchConfigAtom, p.ConfigNotifier, &wg)

				return nil
			},
			OnStop: func(_ context.Context) error {
				wg.Wait()
				return nil
			},
		}),
	}
}

func runDeleter(
	ctx context.Context,
	cm queue.ConsumerMaker,
	runtime *queue.Runtime,
	logger *zap.SugaredLogger,
	esRef *concurrency.AtomicValue[*elasticsearch.Client],
	searchConfigAtom *concurrency.AtomicValue[SearchConfig],
	configNotifier *ConfigNotifier,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	backendCh, unsubscribe := runtime.SubscribeBackendChanges()
	defer unsubscribe()

	cfgCh, unsubscribeCfg := configNotifier.Subscribe()
	defer unsubscribeCfg()

	var lastAddr []string

	handler := func(ctx context.Context, _ string, value []byte) error {
		var infoHash string
		if err := json.Unmarshal(value, &infoHash); err != nil {
			logger.Warnw("failed to unmarshal info hash", "error", err)
			return nil
		}

		if infoHash == "" {
			return nil
		}

		cfg := searchConfigAtom.Get()

		es := esRef.Get()
		if cfg.Backend != backendElasticsearch || es == nil {
			return nil
		}

		if err := es.Delete(ctx, IndexName, infoHash); err != nil {
			logger.Warnw("failed to delete document from ES",
				"info_hash", infoHash,
				"index", IndexName,
				"error", err,
			)
		} else {
			logger.Debugw("deleted document from ES",
				"info_hash", infoHash,
				"index", IndexName,
			)
		}

		return nil
	}

	for {
		consumer, err := cm.NewConsumer(
			kafka.TopicDeleteTorrent,
			"hexmagnet-deleter",
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

		logger.Infow("consumer started", "topic", kafka.TopicDeleteTorrent)

		select {
		case <-backendCh:
			logger.Infow("backend changed, restarting consumer")

			if err := consumer.Stop(ctx); err != nil {
				logger.Errorw("failed to stop consumer", "error", err)
			}
		case <-cfgCh:
			cfg := searchConfigAtom.Get()
			if cfg.Backend != backendElasticsearch {
				esRef.Set(nil)

				lastAddr = nil

				continue
			}

			if reflect.DeepEqual(cfg.Elasticsearch.Addresses, lastAddr) {
				continue
			}

			client, clientErr := elasticsearch.NewClient(cfg.Elasticsearch)
			if clientErr != nil {
				logger.Errorw("failed to create elasticsearch client", "error", clientErr)
				continue
			}

			esRef.Set(client)

			lastAddr = cfg.Elasticsearch.Addresses
			logger.Infow("elasticsearch client updated", "addresses", cfg.Elasticsearch.Addresses)
		case <-ctx.Done():
			if err := consumer.Stop(ctx); err != nil {
				logger.Errorw("failed to stop consumer on shutdown", "error", err)
			}

			return
		}
	}
}
