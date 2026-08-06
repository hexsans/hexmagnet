package dhtcrawler

import (
	"context"
	"errors"
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/server"
)

func NewHealthCheck(
	dhtCrawlerActive *concurrency.AtomicValue[bool],
	lastResponses *concurrency.AtomicValue[server.LastResponses],
) health.Check {
	return health.Check{
		Name: "dht",
		IsActive: func() bool {
			return dhtCrawlerActive.Get()
		},
		Timeout: time.Second,
		Check: func(context.Context) error {
			lr := lastResponses.Get()
			if lr.StartTime.IsZero() {
				return nil
			}

			now := time.Now()
			if lr.LastSuccess.IsZero() {
				if now.Sub(lr.StartTime) < 30*time.Second {
					return nil
				}

				return errors.New("no response within 30 seconds")
			}

			if now.Sub(lr.LastSuccess) > time.Minute {
				return errors.New("no successful responses within last minute")
			}

			return nil
		},
	}
}
