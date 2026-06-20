package persistsources

import (
	"context"
	"encoding/json"

	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/processor"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type QueueConsumerParams struct {
	fx.In
	Handler       Handler
	Producer      queue.Producer
	ConsumerMaker queue.ConsumerMaker
	Logger        *zap.SugaredLogger
	Runtime       *queue.Runtime
}

type QueueConsumerResult struct {
	fx.Out
	Worker worker.Worker `group:"workers"`
}

func NewQueueConsumer(p QueueConsumerParams) QueueConsumerResult {
	return QueueConsumerResult{
		Worker: dht.NewConsumerWorker(
			"dht_persistsources_consumer",
			p.ConsumerMaker,
			p.Runtime,
			kafka.TopicScrapeResult,
			"dht-persistsources",
			func(ctx context.Context, key string, value []byte) error {
				var msg dht.ScrapeResultMessage
				if err := json.Unmarshal(value, &msg); err != nil {
					p.Logger.Warnw("failed to unmarshal persist sources message", "key", key, "error", err)
					return err
				}

				if err := p.Handler.HandlePersistSources(ctx, msg); err != nil {
					p.Logger.Warnw("persist sources handler failed", "info_hash", msg.InfoHash, "error", err)
					return err
				}

				id, err := dht.ParseInfoHash(p.Logger, msg.InfoHash, "persist sources kafka")
				if err != nil {
					return err
				}

				p.Producer.Produce(kafka.TopicProcessTorrent, msg.InfoHash, processor.MessageParams{
					InfoHashes: []protocol.ID{id},
				})

				return nil
			},
			p.Logger.Named("message_queue.persistsources"),
		),
	}
}
