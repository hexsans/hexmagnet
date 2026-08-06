package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/hexsans/hexmagnet/internal/health"
	"go.uber.org/fx"
)

type HealthCheckFxParams struct {
	fx.In
	Runtime *Runtime
}

type HealthCheckFxResult struct {
	fx.Out
	Option health.CheckerOption `group:"health_check_options"`
}

func NewHealthCheckFx(p HealthCheckFxParams) HealthCheckFxResult {
	return HealthCheckFxResult{
		Option: health.WithPeriodicCheck(
			time.Second*30,
			0,
			health.Check{
				Name:    backendKafka,
				Timeout: time.Second * 30,
				IsActive: func() bool {
					return p.Runtime.ActiveBackend.Get() == backendKafka
				},
				Check: func(ctx context.Context) error {
					mgr := p.Runtime.Manager.Get()
					if mgr == nil {
						return fmt.Errorf("kafka manager not initialized")
					}

					_, err := mgr.Jobs(ctx, JobsQuery{Limit: 1})

					return err
				},
			},
		),
	}
}
