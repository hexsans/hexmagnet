package dhtfx

import (
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/client"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/responder"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/server"
	"go.uber.org/fx"
)

func New() fx.Option {
	return fx.Module(
		"dht",
		fx.Provide(
			fx.Annotated{
				Name:   "dht_node_id",
				Target: protocol.RandomNodeIDWithClientSuffix,
			},
			client.New,
			ktable.New,
			responder.New,
			server.New,
		),
	)
}
