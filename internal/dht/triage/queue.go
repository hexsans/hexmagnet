package triage

import (
	"context"
	"encoding/json"

	"github.com/hexsans/hexmagnet/internal/dht"
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
			"dht_triage_consumer",
			p.ConsumerMaker,
			p.Runtime,
			kafka.TopicDiscoveredHashes,
			"dht-triage",
			func(ctx context.Context, key string, value []byte) error {
				var msg dht.DiscoveredHash
				if err := json.Unmarshal(value, &msg); err != nil {
					p.Logger.Warnw("failed to unmarshal triage message", "key", key, "error", err)
					return err
				}

				result, err := p.Handler.HandleTriage(ctx, msg)
				if err != nil {
					p.Logger.Warnw("triage handler failed", "info_hash", msg.InfoHash, "node", msg.Node, "error", err)
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
			p.Logger.Named("message_queue.triage"),
		),
	}
}
