package dhtcrawler

import (
	"context"
	"time"

	bloom "github.com/bits-and-blooms/bloom/v3"
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/client"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Params struct {
	fx.In
	Config          Config
	ConfigManager   *configmgr.Manager `optional:"true"`
	KTable          ktable.Table
	Client          utils.Lazy[client.Client]
	Producer        queue.Producer
	DiscoveredNodes concurrency.BatchingChannel[ktable.Node] `name:"dht_discovered_nodes"`
	Logger          *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Worker worker.Worker `group:"workers"`

	DhtCrawlerActive  *concurrency.AtomicValue[bool] `name:"dht_crawler_active"`
	DhtCrawlerRuntime *Runtime                       `name:"dht_crawler_runtime"`
}

func New(params Params) Result {
	active := &concurrency.AtomicValue[bool]{}
	runtime := NewRuntime()

	var c crawler

	runtime.Active = active

	return Result{
		Worker: worker.NewWorker(
			"dht_crawler",
			fx.Hook{
				OnStart: func(ctx context.Context) error {
					active.Set(true)
					runtime.StartedAt.Set(time.Now())

					scalingFactor := 10
					connLimit := 50

					calcLimit := func(weight int) int {
						l := max(connLimit*weight/101, 1)
						return l
					}

					runtime.NodesForPing = concurrency.NewBufferedConcurrentChannel[ktable.Node](
						scalingFactor, calcLimit(1))
					runtime.NodesForFindNode = concurrency.NewBufferedConcurrentChannel[ktable.Node](
						10*scalingFactor, calcLimit(10))
					runtime.NodesForSampleInfoHashes = concurrency.NewBufferedConcurrentChannel[ktable.Node](
						10*scalingFactor, calcLimit(10))

					cl, err := params.Client.Get()
					if err != nil {
						return err
					}

					c = crawler{
						kTable:                   params.KTable,
						client:                   cl,
						getOldestNodesInterval:   time.Second * 10,
						oldPeerThreshold:         time.Minute * 15,
						discoveredNodes:          params.DiscoveredNodes,
						nodesForPing:             runtime.NodesForPing,
						nodesForFindNode:         runtime.NodesForFindNode,
						nodesForSampleInfoHashes: runtime.NodesForSampleInfoHashes,
						kafkaProducer:            params.Producer,
						ignoreHashes: &ignoreHashes{
							bloom: bloom.NewWithEstimates(10_000_000, 0.001),
						},
						soughtNodeID:    &concurrency.AtomicValue[protocol.ID]{},
						stopped:         make(chan struct{}),
						runtime:         runtime,
						logger:          params.Logger.Named("dht_crawler"),
						nodeDispatchSem: make(chan struct{}, maxConcurrentNodeDispatches),
					}
					c.config.Store(&crawlerConfig{
						hashDiscoverLimiter: rate.NewLimiter(
							rate.Limit(max(params.Config.HashDiscoverLimit, 1)),
							1,
						),
						bootstrapNodes: params.Config.BootstrapNodes,
						reseedInterval: params.Config.ReseedBootstrapNodesInterval,
						embedTrackers:  params.Config.EmbedTrackers,
					})
					c.soughtNodeID.Set(protocol.RandomNodeID())

					if params.ConfigManager != nil {
						params.ConfigManager.Subscribe(ctx, "dht_crawler",
							func(_ context.Context, snap *configmgr.Snapshot) error {
								c.config.Store(&crawlerConfig{
									hashDiscoverLimiter: rate.NewLimiter(
										rate.Limit(snap.DHTRequester.HashDiscoverLimit),
										1,
									),
									bootstrapNodes: snap.DHT.BootstrapNodes,
									reseedInterval: snap.DHT.ReseedBootstrapNodesInterval,
									embedTrackers:  snap.Server.EmbedTrackers,
								})

								return nil
							}, configmgr.ApplyAsync)
					}

					go worker.GoRecover(params.Logger, "crawler_start", func() { c.start(ctx) })

					return nil
				},
				OnStop: func(context.Context) error {
					active.Set(false)

					if c.stopped != nil {
						close(c.stopped)
					}

					return nil
				},
			},
		),
		DhtCrawlerActive:  active,
		DhtCrawlerRuntime: runtime,
	}
}
