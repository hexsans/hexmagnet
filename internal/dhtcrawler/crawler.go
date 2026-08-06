package dhtcrawler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	bloom "github.com/bits-and-blooms/bloom/v3"
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/client"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type crawlerConfig struct {
	hashDiscoverLimiter *rate.Limiter
	bootstrapNodes      []string
	reseedInterval      time.Duration
	embedTrackers       []string
}

type crawler struct {
	kTable                   ktable.Table
	client                   client.Client
	getOldestNodesInterval   time.Duration
	oldPeerThreshold         time.Duration
	discoveredNodes          concurrency.BatchingChannel[ktable.Node]
	nodesForPing             concurrency.BufferedConcurrentChannel[ktable.Node]
	nodesForFindNode         concurrency.BufferedConcurrentChannel[ktable.Node]
	nodesForSampleInfoHashes concurrency.BufferedConcurrentChannel[ktable.Node]
	kafkaProducer            queue.Producer
	ignoreHashes             *ignoreHashes
	soughtNodeID             *concurrency.AtomicValue[protocol.ID]
	stopped                  chan struct{}
	runtime                  *Runtime
	config                   atomic.Pointer[crawlerConfig]
	logger                   *zap.SugaredLogger
	nodeDispatchSem          chan struct{}
}

const maxConcurrentNodeDispatches = 64

func (c *crawler) start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go worker.GoRecover(c.logger, "rotateSoughtNodeID", func() { c.rotateSoughtNodeID(ctx) })
	go worker.GoRecover(c.logger, "runDiscoveredNodes", func() { c.runDiscoveredNodes(ctx) })
	go worker.GoRecover(c.logger, "runPing", func() { c.runPing(ctx) })
	go worker.GoRecover(c.logger, "runFindNode", func() { c.runFindNode(ctx) })
	go worker.GoRecover(c.logger, "getNodesForFindNode", func() { c.getNodesForFindNode(ctx) })
	go worker.GoRecover(c.logger, "runSampleInfoHashes", func() { c.runSampleInfoHashes(ctx) })
	go worker.GoRecover(c.logger, "getNodesForSampleInfoHashes", func() { c.getNodesForSampleInfoHashes(ctx) })
	go worker.GoRecover(c.logger, "reseedBootstrapNodes", func() { c.reseedBootstrapNodes(ctx) })
	go worker.GoRecover(c.logger, "getOldNodes", func() { c.getOldNodes(ctx) })

	select {
	case <-c.stopped:
	case <-ctx.Done():
	}
}

type ignoreHashes struct {
	mutex sync.Mutex
	bloom *bloom.BloomFilter
}

func (i *ignoreHashes) testAndAdd(id protocol.ID) bool {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	return i.bloom.TestAndAdd(id[:])
}

func (c *crawler) rotateSoughtNodeID(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
			c.soughtNodeID.Set(protocol.RandomNodeID())
		}
	}
}
