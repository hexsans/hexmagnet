package persist

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
			"dht_persist_consumer",
			p.ConsumerMaker,
			p.Runtime,
			kafka.TopicMetaInfo,
			"dht-persist",
			func(ctx context.Context, key string, value []byte) error {
				var msg dht.MetaInfoMessage
				if err := json.Unmarshal(value, &msg); err != nil {
					p.Logger.Warnw("failed to unmarshal persist message", "key", key, "error", err)
					return err
				}

				result, err := p.Handler.HandlePersist(ctx, msg)
				if err != nil {
					p.Logger.Warnw("persist handler failed", "info_hash", msg.InfoHash, "error", err)
					return err
				}

				if result.Scrape {
					p.Producer.Produce(kafka.TopicScrape, result.InfoHash, dht.ScrapeMessage{
						InfoHash: msg.InfoHash,
						Node:     msg.Node,
					})
				}

				id, err := dht.ParseInfoHash(p.Logger, result.InfoHash, "persist kafka")
				if err != nil {
					return err
				}

				p.Producer.Produce(kafka.TopicProcessTorrent, result.InfoHash, processor.MessageParams{
					InfoHashes: []protocol.ID{id},
				})

				return nil
			},
			p.Logger.Named("message_queue.persist"),
		),
	}
}
