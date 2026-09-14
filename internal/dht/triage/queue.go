package triage

import (
	"context"
	"encoding/json"

	"github.com/hexsans/hexmagnet/internal/dht"
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
			"dht_triage_consumer",
			kafka.TopicDiscoveredHashes,
			"dht-triage",
			"message_queue.triage",
			func(ctx context.Context, key string, value []byte) error {
				var msg dht.DiscoveredHash
				if err := json.Unmarshal(value, &msg); err != nil {
					p.Logger.Warnw("failed to unmarshal triage message", "key", key, "error", err)
					return permanent.Mark(err)
				}

				result, err := p.Handler.HandleTriage(ctx, msg)
				if err != nil {
					return err
				}

				switch result.Action {
				case ActionGetPeers:
					p.Producer.Produce(kafka.TopicGetPeers, msg.InfoHash, result.GetPeersMsg)
				case ActionScrape:
					p.Producer.Produce(kafka.TopicScrape, msg.InfoHash, result.ScrapeMsg)
				case ActionDiscard:
				}

				return nil
			},
		),
	}
}
