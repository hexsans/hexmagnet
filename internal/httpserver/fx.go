package httpserver

import "go.uber.org/fx"

func NewModule() fx.Option {
	return fx.Module(
		"http_server",
		fx.Provide(
			NewDefaultConfig,
			New,
			NewCors,
		),
	)
}
