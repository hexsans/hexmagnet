package webhook

import (
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	Config Config
	Logger *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Publisher *Publisher
	Worker    worker.Worker `group:"workers"`
}

// New wires the publisher and registers it as a lifecycle worker.
func New(p Params) Result {
	pub := NewPublisher(p.Config, p.Logger.Named("webhook"))

	return Result{
		Publisher: pub,
		Worker: worker.NewWorker("webhook", fx.Hook{
			OnStart: pub.Start,
			OnStop:  pub.Stop,
		}),
	}
}
