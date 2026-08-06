package dhtcrawler

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
)

func (c *crawler) getNodesForSampleInfoHashes(ctx context.Context) {
	for {
		peers := c.kTable.GetNodesForSampleInfoHashes(60)
		for _, p := range peers {
			select {
			case <-ctx.Done():
				return
			case c.nodesForSampleInfoHashes.In() <- p:
				continue
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (c *crawler) runSampleInfoHashes(ctx context.Context) {
	_ = c.nodesForSampleInfoHashes.Run(ctx, func(n ktable.Node) {
		if !n.IsSampleInfoHashesCandidate() {
			return
		}

		res, err := c.client.SampleInfoHashes(ctx, n.Addr(), c.soughtNodeID.Get())
		if err != nil {
			c.logger.Debugw("sample_infohashes failed", "node", n.Addr(), "error", err)
			c.kTable.BatchCommand(
				ktable.DropNode{ID: n.ID(), Reason: fmt.Errorf("sample_infohashes failed: %w", err)},
			)

			return
		}

		var discoveredHashes []string

		for _, s := range res.Samples {
			if !c.ignoreHashes.testAndAdd(s) {
				discoveredHashes = append(discoveredHashes, s.String())
			}
		}

		now := time.Now()

		for _, h := range discoveredHashes {
			if limiter := c.config.Load().hashDiscoverLimiter; limiter != nil {
				if err := limiter.Wait(ctx); err != nil {
					return
				}
			}

			c.kafkaProducer.Produce(kafka.TopicDiscoveredHashes, h, dht.DiscoveredHash{
				InfoHash:     h,
				Node:         n.Addr().String(),
				DiscoveredAt: now,
			})
		}

		if len(discoveredHashes) > 0 {
			c.runtime.PushActivity("discovered",
				fmt.Sprintf("discovered %d hashes from %s", len(discoveredHashes), n.Addr().String()))
		}

		interval := res.Interval
		if len(discoveredHashes) > 0 && interval > 300 {
			interval = 60
		}

		c.kTable.BatchCommand(ktable.PutNode{ID: n.ID(), Addr: n.Addr(), Options: []ktable.NodeOption{
			ktable.NodeResponded(),
			ktable.NodeBep51Support(true),
			ktable.NodeSampleInfoHashesRes(
				len(discoveredHashes),
				res.Num,
				time.Now().Add(time.Duration(interval)*time.Second),
			),
		}})

		if len(res.Nodes) > 0 {
			c.nodeDispatchSem <- struct{}{}

			go func() {
				defer func() { <-c.nodeDispatchSem }()
				defer func() {
					if r := recover(); r != nil {
						c.logger.Errorw("sample_infohashes node send panicked",
							"panic", r,
							"stack", string(debug.Stack()),
						)
					}
				}()

				timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
				defer cancel()

				for _, n := range res.Nodes {
					select {
					case <-timeoutCtx.Done():
						return
					case c.discoveredNodes.In() <- ktable.NewNode(n.ID, n.Addr):
						continue
					}
				}
			}()
		}
	})
}
