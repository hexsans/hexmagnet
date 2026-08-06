package tmdb

import (
	"context"
	"time"

	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/utils"
)

func NewHealthCheck(
	enabled bool,
	client utils.Lazy[Client],
) health.Check {
	return health.Check{
		Name:    "tmdb",
		Timeout: time.Second * 30,
		IsActive: func() bool {
			return enabled
		},
		Check: func(ctx context.Context) error {
			c, err := client.Get()
			if err != nil {
				return err
			}

			return c.ValidateAccessToken(ctx)
		},
	}
}
