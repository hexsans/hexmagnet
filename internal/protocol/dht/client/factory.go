package client

import (
	"time"

	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/server"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Params struct {
	fx.In
	NodeID protocol.ID `name:"dht_node_id"`
	Server utils.Lazy[server.Server]
	Logger *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Client utils.Lazy[Client]
}

func New(p Params) Result {
	return Result{
		Client: utils.NewLazy(func() (Client, error) {
			s, err := p.Server.Get()
			if err != nil {
				return nil, err
			}

			return clientLogger{
				client: serverAdapter{
					nodeID: p.NodeID,
					server: s,
				},
				// we make way to many queries to usefully log everything, but having a sample is
				// helpful:
				logger: p.Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
					return zapcore.NewSamplerWithOptions(core, time.Minute, 10, 0)
				})).Named("dht_client"),
			}, nil
		}),
	}
}
