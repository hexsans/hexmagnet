package worker

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewModule() fx.Option {
	return fx.Module(
		"worker",
		fx.Provide(NewRegistry),
		fx.Invoke(func(lc fx.Lifecycle, reg Registry, logger *zap.SugaredLogger) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					reg.EnableAll()
					logger.Infow("starting all workers")

					return reg.Start(ctx)
				},
				OnStop: func(ctx context.Context) error {
					logger.Infow("stopping all workers")
					return reg.Stop(ctx)
				},
			})
		}),
	)
}
