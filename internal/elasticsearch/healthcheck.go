package elasticsearch

import (
	"context"
	"time"

	"github.com/hexsans/hexmagnet/internal/health"
)

// NewHealthCheck builds the elasticsearch health check.
// The check is only active while `active` reports true (evaluated on every
// run), and it probes a fresh client produced by `newClient` on each run so
// that runtime config changes (backend or address switches) are reflected
// immediately.
func NewHealthCheck(
	active func() bool,
	newClient func() (*Client, error),
) health.Check {
	return health.Check{
		Name:    "elasticsearch",
		Timeout: time.Second * 10,
		IsActive: func() bool {
			return active()
		},
		Check: func(ctx context.Context) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			return client.Ping(ctx)
		},
	}
}
