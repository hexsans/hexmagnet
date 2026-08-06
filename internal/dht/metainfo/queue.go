package metainfo

import (
	"context"
	"encoding/json"

	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

const maxConcurrentGetPeers = 50

var semGetPeers = semaphore.NewWeighted(maxConcurrentGetPeers)

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
			"dht_metainfo_consumer",
			p.ConsumerMaker,
			p.Runtime,
			kafka.TopicGetPeers,
			"dht-metainfo",
			func(ctx context.Context, _ string, value []byte) error {
				var msg dht.GetPeersMessage
				if err := json.Unmarshal(value, &msg); err != nil {
					return err
				}

				if err := semGetPeers.Acquire(ctx, 1); err != nil {
					return err
				}

				worker.GoRecover(p.Logger.Named("message_queue.metainfo.handler"), "handle_get_peers", func() {
					defer semGetPeers.Release(1)

					mi, err := p.Handler.HandleGetPeers(ctx, msg)
					if err != nil {
						p.Logger.Warnw("metainfo get_peers failed", "error", err)
						return
					}

					if mi == nil {
						return
					}

					p.Producer.Produce(kafka.TopicMetaInfo, msg.InfoHash, mi)
				})

				return nil
			},
			p.Logger.Named("message_queue.metainfo"),
		),
	}
}
