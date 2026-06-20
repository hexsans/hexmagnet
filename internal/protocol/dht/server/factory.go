package server

import (
	"context"
	"fmt"
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/responder"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Params struct {
	fx.In
	Config         Config
	Responder      responder.Responder
	Logger         *zap.SugaredLogger
	RequestLimiter *rate.Limiter `name:"global_request_limiter" optional:"true"`
}

type Result struct {
	fx.Out
	Server            utils.Lazy[Server]
	ResponderEnabled  *atomic.Bool
	LastResponses     *concurrency.AtomicValue[LastResponses] `name:"dht_server_last_responses"`
	AppHook           fx.Hook                                 `                                 group:"app_hooks"`
	QueryDuration     prometheus.Collector                    `                                 group:"prometheus_collectors"`
	QuerySuccessTotal prometheus.Collector                    `                                 group:"prometheus_collectors"`
	QueryErrorTotal   prometheus.Collector                    `                                 group:"prometheus_collectors"`
	QueryConcurrency  prometheus.Collector                    `                                 group:"prometheus_collectors"`
}

const (
	namespace = "hexmagnet"
	subsystem = "dht_server"
)

func New(p Params) Result {
	lastResponses := &concurrency.AtomicValue[LastResponses]{}
	responderEnabled := &atomic.Bool{}
	responderEnabled.Store(p.Config.ResponderEnabled)

	collector := newPrometheusCollector()
	ls := utils.NewLazy(func() (Server, error) {
		base := queryLimiter{
			server: prometheusServerWrapper{
				prometheusCollector: collector,
				server: healthCollector{
					baseServer: &server{
						stopped: make(chan struct{}),
						localAddr: netip.AddrPortFrom(
							netip.IPv4Unspecified(),
							p.Config.Port,
						),
						socket:           NewSocket(),
						queries:          make(map[string]chan dht.RecvMsg),
						queryTimeout:     5 * time.Second,
						responder:        p.Responder,
						responderTimeout: time.Second * 5,
						responderEnabled: responderEnabled,
						idIssuer:         &variantIDIssuer{},
						logger:           p.Logger.Named(subsystem),
					},
					lastResponses: lastResponses,
				},
			},
			queryLimiter: concurrency.NewKeyedLimiter(rate.Every(time.Second), 4, 1000, time.Second*20),
		}

		var s Server = &base
		if p.RequestLimiter != nil {
			s = globalRequestRateLimiter{
				server:  s,
				limiter: p.RequestLimiter,
			}
		}

		if err := s.start(); err != nil {
			return nil, fmt.Errorf("could not start server: %w", err)
		}

		return s, nil
	})

	return Result{
		Server:           ls,
		ResponderEnabled: responderEnabled,
		AppHook: fx.Hook{
			OnStop: func(context.Context) error {
				return ls.IfInitialized(func(s Server) error {
					s.stop()
					return nil
				})
			},
		},
		LastResponses:     lastResponses,
		QueryDuration:     collector.queryDuration,
		QuerySuccessTotal: collector.querySuccessTotal,
		QueryErrorTotal:   collector.queryErrorTotal,
		QueryConcurrency:  collector.queryConcurrency,
	}
}
