package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/queue/supervisor"
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

	esRef := &concurrency.AtomicValue[*elasticsearch.Client]{}
	esRef.Set(p.ESClient)

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

		cfg := p.SearchConfigAtom.Get()

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

			return fmt.Errorf("delete torrent %s from index: %w", infoHash, err)
		}

		logger.Debugw("deleted document from ES",
			"info_hash", infoHash,
			"index", IndexName,
		)

		return nil
	}

	sup := supervisor.New(supervisor.Params{
		Maker:   p.ConsumerMaker,
		Runtime: p.Runtime,
		Topic:   kafka.TopicDeleteTorrent,
		GroupID: "hexmagnet-deleter",
		Handler: handler,
		Trigger: &supervisor.Trigger{
			Subscribe: p.ConfigNotifier.Subscribe,
			OnEvent: func(context.Context) bool {
				cfg := p.SearchConfigAtom.Get()
				if cfg.Backend != backendElasticsearch {
					esRef.Set(nil)

					lastAddr = nil

					return false
				}

				if reflect.DeepEqual(cfg.Elasticsearch.Addresses, lastAddr) {
					return false
				}

				client, clientErr := elasticsearch.NewClient(cfg.Elasticsearch)
				if clientErr != nil {
					logger.Errorw("failed to create elasticsearch client", "error", clientErr)

					return false
				}

				esRef.Set(client)

				lastAddr = cfg.Elasticsearch.Addresses
				logger.Infow("elasticsearch client updated", "addresses", cfg.Elasticsearch.Addresses)

				return false
			},
		},
		Logger: logger,
	})

	return Result{
		Worker: worker.NewWorker("message_queue_enrich_deleter", sup.Hook()),
	}
}
