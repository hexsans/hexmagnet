package processor

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/queue/permanent"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ConsumerParams struct {
	fx.In
	ConsumerMaker queue.ConsumerMaker
	Logger        *zap.SugaredLogger
	Processor     utils.Lazy[Processor]
	Runtime       *queue.Runtime
}

type ConsumerResult struct {
	fx.Out
	Worker worker.Worker `group:"workers"`
}

func NewConsumer(p ConsumerParams) ConsumerResult {
	logger := p.Logger.Named("message_queue.process")

	var wg sync.WaitGroup

	return ConsumerResult{
		Worker: worker.NewWorker("message_queue_process_consumer", fx.Hook{
			OnStart: func(ctx context.Context) error {
				wg.Add(1)
				go runManagedConsumer(ctx, p.ConsumerMaker, p.Runtime, logger, p.Processor, &wg)

				return nil
			},
			OnStop: func(_ context.Context) error {
				wg.Wait()
				return nil
			},
		}),
	}
}

func runManagedConsumer(
	ctx context.Context,
	cm queue.ConsumerMaker,
	runtime *queue.Runtime,
	logger *zap.SugaredLogger,
	proc utils.Lazy[Processor],
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	backendCh, unsubscribe := runtime.SubscribeBackendChanges()
	defer unsubscribe()

	pr, err := proc.Get()
	if err != nil {
		logger.Errorw("failed to get processor", "error", err)
		return
	}

	handler := func(ctx context.Context, _ string, value []byte) error {
		msg := &MessageParams{}
		if err := json.Unmarshal(value, msg); err != nil {
			logger.Errorw("failed to unmarshal message", "error", err)
			return permanent.Mark(err)
		}

		return pr.Process(ctx, *msg)
	}

	for {
		consumer, err := cm.NewConsumer(
			kafka.TopicProcessTorrent,
			"hexmagnet-processor",
			handler,
			logger,
		)
		if err != nil {
			logger.Errorw("failed to create consumer", "error", err)

			select {
			case <-time.After(time.Second):
				continue
			case <-ctx.Done():
				return
			}
		}

		if err := consumer.Start(ctx); err != nil {
			logger.Errorw("failed to start consumer", "error", err)

			select {
			case <-time.After(time.Second):
				continue
			case <-ctx.Done():
				return
			}
		}

		select {
		case <-backendCh:
			logger.Infow("backend changed, restarting consumer")

			if err := consumer.Stop(ctx); err != nil {
				logger.Errorw("failed to stop consumer", "error", err)
			}
		case <-ctx.Done():
			if err := consumer.Stop(ctx); err != nil {
				logger.Errorw("failed to stop consumer on shutdown", "error", err)
			}

			return
		}
	}
}
