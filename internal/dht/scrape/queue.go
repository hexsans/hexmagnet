package scrape

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

const maxConcurrentScrape = 10

var semScrape = semaphore.NewWeighted(maxConcurrentScrape)

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
			"dht_scrape_consumer",
			p.ConsumerMaker,
			p.Runtime,
			kafka.TopicScrape,
			"dht-scrape",
			func(ctx context.Context, _ string, value []byte) error {
				var msg dht.ScrapeMessage
				if err := json.Unmarshal(value, &msg); err != nil {
					return err
				}

				if err := semScrape.Acquire(ctx, 1); err != nil {
					return err
				}

				worker.GoRecover(p.Logger.Named("message_queue.scrape.handler"), "handle_scrape", func() {
					defer semScrape.Release(1)

					result, err := p.Handler.HandleScrape(ctx, msg)
					if err != nil {
						p.Logger.Warnw("scrape failed", "error", err)
						return
					}

					if result == nil {
						return
					}

					p.Producer.Produce(kafka.TopicScrapeResult, msg.InfoHash, result)
				})

				return nil
			},
			p.Logger.Named("message_queue.scrape"),
		),
	}
}
