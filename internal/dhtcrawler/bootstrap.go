package dhtcrawler

import (
	"context"
	"net"
	"time"

	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
)

func (c *crawler) reseedBootstrapNodes(ctx context.Context) {
	interval := time.Duration(0)

	for {
		cfg := c.config.Load()

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			for _, strAddr := range cfg.bootstrapNodes {
				addr, err := net.ResolveUDPAddr("udp", strAddr)
				if err != nil {
					c.logger.Warnw("failed to resolve bootstrap node address",
						"addr", strAddr,
						"error", err,
					)

					continue
				}

				select {
				case <-ctx.Done():
					return
				case c.nodesForPing.In() <- ktable.NewNode(ktable.ID{}, addr.AddrPort()):
					continue
				}
			}

			c.logger.Infow("seeding routing table with bootstrap addresses",
				"addresses", cfg.bootstrapNodes,
			)
		}

		interval = cfg.reseedInterval
	}
}
