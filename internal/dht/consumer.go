package dht

import (
	"context"

	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/supervisor"
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
		sup := supervisor.New(supervisor.Params{
			Maker:   consumerMaker,
			Runtime: runtime,
			Topic:   topic,
			GroupID: groupID,
			Handler: handler,
			Logger:  logger,
		})

		return worker.NewWorker(name, sup.Hook())
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
