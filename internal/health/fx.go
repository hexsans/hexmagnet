package health

import "go.uber.org/fx"

func NewModule() fx.Option {
	return fx.Module(
		"health",
		fx.Provide(
			New,
		),
	)
}
