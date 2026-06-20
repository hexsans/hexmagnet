package dhtcrawler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
)

var ErrMismatchingNodeID = errors.New("node responded with a mismatching ID")

func (c *crawler) runPing(ctx context.Context) {
	_ = c.nodesForPing.Run(ctx, func(n ktable.Node) {
		if n.Dropped() || n.Time().After(time.Now().Add(-c.oldPeerThreshold)) {
			return
		}

		res, err := c.client.Ping(ctx, n.Addr())

		var nodeID protocol.ID

		if err == nil {
			nodeID = res.ID
			if !n.ID().IsZero() && n.ID() != nodeID {
				c.logger.Debugw(
					"node identity changed, removing stale routing table entry",
					"expected",
					n.ID(),
					"got",
					res.ID,
					"node",
					n.Addr(),
				)
				nodeID = n.ID()
				err = ErrMismatchingNodeID
			}
		}

		if err != nil {
			if !errors.Is(err, ErrMismatchingNodeID) {
				c.logger.Debugw("ping failed", "node", n.Addr(), "error", err)
			}

			c.kTable.BatchCommand(ktable.DropNode{
				ID:     nodeID,
				Reason: fmt.Errorf("failed to respond to ping: %w", err),
			})
		} else {
			if !c.runtime.SeenConnectedPeers.TestAndAdd(n.Addr().Addr()) {
				c.runtime.PeersConnected.Update(func(v uint64) uint64 { return v + 1 })
			}

			c.kTable.BatchCommand(ktable.PutNode{
				ID:      nodeID,
				Addr:    n.Addr(),
				Options: []ktable.NodeOption{ktable.NodeResponded()},
			},
			)
		}
	})
}

// getOldNodes periodically adds the oldest nodes from the routing table to the nodesForPing channel,
// so they can be pruned from the routing table if no longer responsive.
func (c *crawler) getOldNodes(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(c.getOldestNodesInterval):
			for _, p := range c.kTable.GetOldestNodes(time.Now().Add(-c.oldPeerThreshold), 0) {
				select {
				case <-ctx.Done():
					return
				case c.nodesForPing.In() <- p:
					continue
				}
			}
		}
	}
}
