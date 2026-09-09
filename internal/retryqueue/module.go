package retryqueue

import (
	"go.uber.org/fx"
)

func NewModule() fx.Option {
	return fx.Module(
		"retryqueue",
		fx.Provide(
			NewQueue,
			NewScheduler,
		),
	)
}
