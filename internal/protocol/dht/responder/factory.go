package responder

import (
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
)

type Config struct {
	GlobalRateLimit int
	PerIPRateLimit  int
}

type Params struct {
	fx.In
	KTable          ktable.Table
	DiscoveredNodes concurrency.BatchingChannel[ktable.Node] `name:"dht_discovered_nodes"`
	Logger          *zap.SugaredLogger
	Config          Config
}

type Result struct {
	fx.Out
	Responder         Responder
	RateLimiter       Limiter
	QueryDuration     prometheus.Collector `group:"prometheus_collectors"`
	QuerySuccessTotal prometheus.Collector `group:"prometheus_collectors"`
	QueryErrorTotal   prometheus.Collector `group:"prometheus_collectors"`
	QueryConcurrency  prometheus.Collector `group:"prometheus_collectors"`
}

const (
	namespace = "hexmagnet"
	subsystem = "dht_responder"
)

func New(p Params) Result {
	globalRate := rate.Inf
	if p.Config.GlobalRateLimit > 0 {
		globalRate = rate.Limit(p.Config.GlobalRateLimit)
	}

	perIPRate := rate.Inf
	if p.Config.PerIPRateLimit > 0 {
		perIPRate = rate.Limit(p.Config.PerIPRateLimit)
	}

	globalBurst := p.Config.GlobalRateLimit
	if globalBurst <= 0 {
		globalBurst = 20
	}

	perIPBurst := p.Config.PerIPRateLimit
	if perIPBurst <= 0 {
		perIPBurst = 10
	}

	limiter := NewLimiter(globalRate, globalBurst, perIPRate, perIPBurst, 1000, time.Second*20)

	collector := newPrometheusCollector(responderLimiter{
		responder: responder{
			nodeID:                   p.KTable.Origin(),
			kTable:                   p.KTable,
			tokenSecret:              protocol.RandomNodeID().Bytes(),
			sampleInfoHashesInterval: 10,
		},
		limiter: limiter,
	})

	return Result{
		Responder: responderNodeDiscovery{
			responder: responderLogger{
				responder: collector,
				logger: p.Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
					return zapcore.NewSamplerWithOptions(core, time.Minute, 10, 0)
				})).Named(subsystem),
			},
			discoveredNodes: p.DiscoveredNodes.In(),
		},
		RateLimiter:       limiter,
		QueryDuration:     collector.queryDuration,
		QuerySuccessTotal: collector.querySuccessTotal,
		QueryErrorTotal:   collector.queryErrorTotal,
		QueryConcurrency:  collector.queryConcurrency,
	}
}
