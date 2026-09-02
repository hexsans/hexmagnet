package metainforequester

import (
	"net"
	"time"

	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

type Params struct {
	fx.In
	Config         Config
	Logger         *zap.SugaredLogger
	RequestLimiter *rate.Limiter `name:"global_request_limiter" optional:"true"`
}

type Result struct {
	fx.Out
	Requester Requester
}

func New(p Params) Result {
	base := requestLimiter{
		requester: requestLogger{
			requester: requester{
				clientID: protocol.RandomPeerID(),
				timeout:  5 * time.Second,
				dialer: &net.Dialer{
					Timeout:   3 * time.Second,
					KeepAlive: -1,
				},
			},
			logger: p.Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				return zapcore.NewSamplerWithOptions(core, time.Minute, 10, 0)
			})).Named("meta_info_requester"),
		},
		limiter: concurrency.NewKeyedLimiter(rate.Every(time.Second/2), 4, 1000, time.Second*20),
	}

	var req Requester = &base
	if p.RequestLimiter != nil {
		req = &requestRateLimiter{
			requester: req,
			limiter:   p.RequestLimiter,
		}
	}

	req = &connectionSemaphore{
		requester: req,
		sem:       semaphore.NewWeighted(100),
	}

	return Result{
		Requester: req,
	}
}
