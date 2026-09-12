package dht

import (
	"context"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewConsumerWorker(
	name string,
	consumerMaker queue.ConsumerMaker,
	runtime *queue.Runtime,
	topic string,
	groupID string,
	handler queue.MessageHandler,
	logger *zap.SugaredLogger,
) worker.Worker {
	if runtime != nil {
		var wg sync.WaitGroup

		return worker.NewWorker(name, fx.Hook{
			OnStart: func(ctx context.Context) error {
				wg.Add(1)
				go func() {
					defer wg.Done()

					backendCh, unsubscribe := runtime.SubscribeBackendChanges()
					defer unsubscribe()

					for {
						consumer, err := consumerMaker.NewConsumer(topic, groupID, handler, logger)
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
							logger.Infow("backend changed, restarting consumer", "name", name)

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
				}()

				return nil
			},
			OnStop: func(context.Context) error {
				wg.Wait()
				return nil
			},
		})
	}

	var consumer queue.Consumer

	return worker.NewWorker(name, fx.Hook{
		OnStart: func(ctx context.Context) error {
			var err error

			consumer, err = consumerMaker.NewConsumer(topic, groupID, handler, logger)
			if err != nil {
				return err
			}

			return consumer.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return consumer.Stop(ctx)
		},
	})
}
