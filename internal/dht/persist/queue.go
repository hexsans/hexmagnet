package persist

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
			"dht_persist_consumer",
			kafka.TopicMetaInfo,
			"dht-persist",
			"message_queue.persist",
			func(ctx context.Context, key string, value []byte) error {
				var msg dht.MetaInfoMessage
				if err := json.Unmarshal(value, &msg); err != nil {
					p.Logger.Warnw("failed to unmarshal persist message", "key", key, "error", err)
					return permanent.Mark(err)
				}

				result, err := p.Handler.HandlePersist(ctx, msg)
				if err != nil {
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
		),
	}
}
