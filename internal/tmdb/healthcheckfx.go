package tmdb

import (
	"time"

	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
)

type HealthCheckFxParams struct {
	fx.In
	Config Config
	Client utils.Lazy[Client]
}

type HealthCheckFxResult struct {
	fx.Out
	Option health.CheckerOption `group:"health_check_options"`
}

func NewHealthCheckFx(p HealthCheckFxParams) HealthCheckFxResult {
	return HealthCheckFxResult{
		Option: health.WithPeriodicCheck(
			time.Minute*5,
			0,
			NewHealthCheck(p.Config.Enabled, p.Client),
		),
	}
}
