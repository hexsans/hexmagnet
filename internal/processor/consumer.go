package processor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/queue/permanent"
	"github.com/hexsans/hexmagnet/internal/queue/supervisor"
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

	sup := supervisor.New(supervisor.Params{
		Maker:   p.ConsumerMaker,
		Runtime: p.Runtime,
		Topic:   kafka.TopicProcessTorrent,
		GroupID: "hexmagnet-processor",
		Prepare: func(context.Context) (queue.MessageHandler, error) {
			pr, err := p.Processor.Get()
			if err != nil {
				return nil, fmt.Errorf("get processor: %w", err)
			}

			return func(ctx context.Context, _ string, value []byte) error {
				msg := &MessageParams{}
				if err := json.Unmarshal(value, msg); err != nil {
					logger.Errorw("failed to unmarshal message", "error", err)

					return permanent.Mark(err)
				}

				return pr.Process(ctx, *msg)
			}, nil
		},
		Logger: logger,
	})

	return ConsumerResult{
		Worker: worker.NewWorker("message_queue_process_consumer", sup.Hook()),
	}
}
