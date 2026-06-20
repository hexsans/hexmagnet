package database

import (
	"context"
	"fmt"
	"time"

	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

type HealthCheckParams struct {
	fx.In
	Pool utils.Lazy[*pgxpool.Pool]
}

type HealthCheckResult struct {
	fx.Out
	Option health.CheckerOption `group:"health_check_options"`
}

func NewHealthCheck(p HealthCheckParams) HealthCheckResult {
	return HealthCheckResult{
		Option: health.WithPeriodicCheck(
			time.Second*30,
			time.Second*1,
			health.Check{
				Name:    "postgres",
				Timeout: time.Second * 5,
				Check: func(ctx context.Context) error {
					pool, poolErr := p.Pool.Get()
					if poolErr != nil {
						return fmt.Errorf("failed to get database pool: %w", poolErr)
					}

					pingErr := pool.Ping(ctx)
					if pingErr != nil {
						return fmt.Errorf("failed to ping database: %w", pingErr)
					}

					return nil
				},
			}),
	}
}
