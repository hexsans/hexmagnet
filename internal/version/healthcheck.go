package version

import (
	"github.com/hexsans/hexmagnet/internal/health"
	"go.uber.org/fx"
)

type HealthCheckResult struct {
	fx.Out
	HealthOption health.CheckerOption `group:"health_check_options"`
}

func NewHealthCheck() HealthCheckResult {
	return HealthCheckResult{
		HealthOption: health.WithInfo(map[string]any{
			"name":    "hexmagnet",
			"version": GitTagValue(),
		}),
	}
}
