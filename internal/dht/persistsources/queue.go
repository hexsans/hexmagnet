package persistsources

import (
	"context"
	"encoding/json"

	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/processor"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/queue/permanent"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
)

type QueueConsumerParams struct {
	fx.In
	dht.QueueConsumerParams

	Handler  Handler
	Producer queue.Producer
}

type QueueConsumerResult struct {
	fx.Out
	Worker worker.Worker `group:"workers"`
}

func NewQueueConsumer(p QueueConsumerParams) QueueConsumerResult {
	return QueueConsumerResult{
		Worker: p.NewWorker(
			"dht_persistsources_consumer",
			kafka.TopicScrapeResult,
			"dht-persistsources",
			"message_queue.persistsources",
			func(ctx context.Context, key string, value []byte) error {
				var msg dht.ScrapeResultMessage
				if err := json.Unmarshal(value, &msg); err != nil {
					p.Logger.Warnw("failed to unmarshal persist sources message", "key", key, "error", err)
					return permanent.Mark(err)
				}

				if err := p.Handler.HandlePersistSources(ctx, msg); err != nil {
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
		),
	}
}
