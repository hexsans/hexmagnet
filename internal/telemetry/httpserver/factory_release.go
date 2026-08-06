//go:build !debug

package httpserver

import (
	"github.com/hexsans/hexmagnet/internal/httpserver"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	PrometheusRegistry utils.Lazy[*prometheus.Registry]
}

type Result struct {
	fx.Out
	PrometheusOption httpserver.Option `group:"http_server_options"`
}

func New(p Params) Result {
	return Result{
		PrometheusOption: prometheusBuilder{
			registry: p.PrometheusRegistry,
		},
	}
}
