package metainfo

import (
	"context"
	"encoding/json"

	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/queue/permanent"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"golang.org/x/sync/semaphore"
)

const maxConcurrentGetPeers = 50

var semGetPeers = semaphore.NewWeighted(maxConcurrentGetPeers)

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
			"dht_metainfo_consumer",
			kafka.TopicGetPeers,
			"dht-metainfo",
			"message_queue.metainfo",
			func(ctx context.Context, _ string, value []byte) error {
				var msg dht.GetPeersMessage
				if err := json.Unmarshal(value, &msg); err != nil {
					return permanent.Mark(err)
				}

				if err := semGetPeers.Acquire(ctx, 1); err != nil {
					return err
				}

				worker.GoRecover(p.Logger.Named("message_queue.metainfo.handler"), "handle_get_peers", func() {
					defer semGetPeers.Release(1)

					mi, err := p.Handler.HandleGetPeers(ctx, msg)
					if err != nil {
						return
					}

					if mi == nil {
						return
					}

					p.Producer.Produce(kafka.TopicMetaInfo, msg.InfoHash, mi)
				})

				return nil
			},
		),
	}
}
