package dhtcrawler

import (
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/server"
	"go.uber.org/fx"
)

type HealthCheckFxParams struct {
	fx.In
	DhtCrawlerActive       *concurrency.AtomicValue[bool]                 `name:"dht_crawler_active"`
	DhtServerLastResponses *concurrency.AtomicValue[server.LastResponses] `name:"dht_server_last_responses"`
}

type HealthCheckFxResult struct {
	fx.Out
	Option health.CheckerOption `group:"health_check_options"`
}

func NewHealthCheckFx(params HealthCheckFxParams) HealthCheckFxResult {
	return HealthCheckFxResult{
		Option: health.WithPeriodicCheck(
			time.Second*10,
			time.Second*1,
			NewHealthCheck(params.DhtCrawlerActive, params.DhtServerLastResponses),
		),
	}
}
