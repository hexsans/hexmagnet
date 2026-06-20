package telemetry

import (
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Collectors []prometheus.Collector `group:"prometheus_collectors"`
}

type Result struct {
	fx.Out
	Registry utils.Lazy[*prometheus.Registry]
}

const Namespace = "hexmagnet"

func New(p Params) (Result, error) {
	return Result{
		Registry: utils.NewLazy[*prometheus.Registry](func() (*prometheus.Registry, error) {
			registry := prometheus.NewRegistry()

			cs := append(
				[]prometheus.Collector{
					collectors.NewGoCollector(),
					collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
						Namespace: Namespace,
					}),
				},
				p.Collectors...,
			)
			for _, c := range cs {
				if err := registry.Register(c); err != nil {
					return nil, err
				}
			}

			return registry, nil
		}),
	}, nil
}
