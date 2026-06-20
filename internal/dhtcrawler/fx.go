package dhtcrawler

import (
	"net"
	"net/netip"

	adht "github.com/anacrolix/dht/v2"
	"go.uber.org/fx"
)

func NewModule() fx.Option {
	return fx.Module(
		"dht_crawler",
		fx.Provide(
			fx.Annotated{
				Name: "dht_bootstrap_nodes",
				Target: func() []netip.AddrPort {
					addrs := make([]netip.AddrPort, 0, len(adht.DefaultGlobalBootstrapHostPorts))
					for _, strAddr := range adht.DefaultGlobalBootstrapHostPorts {
						addr, err := net.ResolveUDPAddr("udp", strAddr)
						if err != nil {
							panic(err)
						}

						addrs = append(addrs, addr.AddrPort())
					}

					return addrs
				},
			},
			New,
			NewDiscoveredNodes,
			NewHealthCheckFx,
		),
		NewRuntimeStoreFxModule(),
	)
}
