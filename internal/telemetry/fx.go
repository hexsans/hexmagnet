package telemetry

import (
	"github.com/hexsans/hexmagnet/internal/telemetry/httpserver"
	"go.uber.org/fx"
)

func NewModule() fx.Option {
	return fx.Module(
		"telemetry",
		fx.Provide(
			httpserver.New,
			New,
		),
	)
}
