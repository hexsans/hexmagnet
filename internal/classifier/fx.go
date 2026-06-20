package classifier

import "go.uber.org/fx"

func NewModule() fx.Option {
	return fx.Module(
		"workflow",
		fx.Provide(
			New,
		),
	)
}
