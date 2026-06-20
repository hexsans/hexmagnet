package dhtcrawler

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	bloomv3 "github.com/bits-and-blooms/bloom/v3"
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/health"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/client"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
	ktable_mocks "github.com/hexsans/hexmagnet/internal/protocol/dht/ktable/mocks"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/server"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type mockClient struct {
	pingErr             error
	findNodeErr         error
	sampleInfoHashesErr error
}

var _ client.Client = (*mockClient)(nil)

func (m *mockClient) Ping(_ context.Context, _ netip.AddrPort) (client.PingResult, error) {
	return client.PingResult{}, m.pingErr
}

func (m *mockClient) FindNode(_ context.Context, _ netip.AddrPort, _ protocol.ID) (client.FindNodeResult, error) {
	return client.FindNodeResult{}, m.findNodeErr
}

func (*mockClient) GetPeers(_ context.Context, _ netip.AddrPort, _ protocol.ID) (client.GetPeersResult, error) {
	return client.GetPeersResult{}, nil
}

func (*mockClient) GetPeersScrape(_ context.Context, _ netip.AddrPort, _ protocol.ID) (client.GetPeersScrapeResult, error) {
	return client.GetPeersScrapeResult{}, nil
}

func (m *mockClient) SampleInfoHashes(_ context.Context, _ netip.AddrPort, _ protocol.ID) (client.SampleInfoHashesResult, error) {
	return client.SampleInfoHashesResult{}, m.sampleInfoHashesErr
}

type mockProducer struct{}

var _ queue.Producer = (*mockProducer)(nil)

func (*mockProducer) Produce(_ string, _ string, _ any) {}
func (*mockProducer) Close() error                      { return nil }

func newMinimalCrawler(t *testing.T) *crawler {
	t.Helper()

	kt := ktable_mocks.NewTable(t)
	kt.On("GetOldestNodes", mock.Anything, mock.Anything).Return([]ktable.Node{}).Maybe()
	kt.On("GetNodesForSampleInfoHashes", mock.Anything).Return([]ktable.Node{}).Maybe()
	kt.On("FilterKnownAddrs", mock.Anything).Return([]netip.Addr{}).Maybe()

	c := &crawler{
		kTable:                   kt,
		client:                   &mockClient{},
		getOldestNodesInterval:   time.Hour,
		oldPeerThreshold:         time.Minute * 15,
		discoveredNodes:          concurrency.NewBatchingChannel[ktable.Node](10, 10, time.Hour),
		nodesForPing:             concurrency.NewBufferedConcurrentChannel[ktable.Node](10, 1),
		nodesForFindNode:         concurrency.NewBufferedConcurrentChannel[ktable.Node](10, 1),
		nodesForSampleInfoHashes: concurrency.NewBufferedConcurrentChannel[ktable.Node](10, 1),
		kafkaProducer:            &mockProducer{},
		ignoreHashes: &ignoreHashes{
			bloom: bloomv3.NewWithEstimates(1000, 0.01),
		},
		soughtNodeID: &concurrency.AtomicValue[protocol.ID]{},
		stopped:      make(chan struct{}),
		runtime: &Runtime{
			SeenConnectedPeers:  newSeenPeersBloom(),
			SeenDiscoveredPeers: newSeenPeersBloom(),
		},
		logger: zap.NewNop().Sugar(),
	}
	c.config.Store(&crawlerConfig{
		hashDiscoverLimiter: rate.NewLimiter(rate.Inf, 1),
		bootstrapNodes:      nil,
		reseedInterval:      time.Hour,
	})

	return c
}

func TestNewCrawlerFactory(t *testing.T) {
	t.Parallel()

	kt := ktable_mocks.NewTable(t)

	params := Params{
		Config: Config{
			BootstrapNodes:               defaultBootstrapNodes,
			ReseedBootstrapNodesInterval: time.Minute,
			RescrapeThreshold:            2592000,
			HashDiscoverLimit:            10,
			EmbedTrackers:                []string{},
		},
		KTable:          kt,
		Client:          utils.NewLazy(func() (client.Client, error) { return &mockClient{}, nil }),
		Producer:        &mockProducer{},
		DiscoveredNodes: concurrency.NewBatchingChannel[ktable.Node](10, 10, time.Hour),
		Logger:          zap.NewNop().Sugar(),
	}

	result := New(params)
	require.NotNil(t, result.Worker)
	require.NotNil(t, result.DhtCrawlerActive)
	require.NotNil(t, result.DhtCrawlerRuntime)
	assert.NotNil(t, result.DhtCrawlerRuntime.SeenConnectedPeers)
	assert.NotNil(t, result.DhtCrawlerRuntime.SeenDiscoveredPeers)
}

func TestNewStore(t *testing.T) {
	t.Parallel()

	runtime := NewRuntime()
	store := NewStore(nil, runtime, zap.NewNop().Sugar())
	require.NotNil(t, store)
	assert.Equal(t, runtime, store.runtime)
	assert.NotNil(t, store.stop)
}

func TestNewRuntimeStoreFxModule(t *testing.T) {
	t.Parallel()

	module := NewRuntimeStoreFxModule()
	assert.NotNil(t, module)
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	module := NewModule()
	assert.NotNil(t, module)
}

func TestNewHealthCheckFx(t *testing.T) {
	t.Parallel()

	active := &concurrency.AtomicValue[bool]{}
	active.Set(true)

	lastResponses := &concurrency.AtomicValue[server.LastResponses]{}

	params := HealthCheckFxParams{
		DhtCrawlerActive:       active,
		DhtServerLastResponses: lastResponses,
	}

	result := NewHealthCheckFx(params)
	require.NotNil(t, result.Option)

	var opt health.CheckerOption
	assert.IsType(t, opt, result.Option)
}

func TestCrawler_RotateSoughtNodeID_DoesNotChangeBeforeTimer(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	c.soughtNodeID.Set(testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.rotateSoughtNodeID(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("rotateSoughtNodeID did not return after context cancellation")
	}

	assert.Equal(t,
		testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		c.soughtNodeID.Get(),
		"node ID should not change because timer hadn't fired",
	)
}

func TestCrawler_RotateSoughtNodeID_CancelledContext(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})

	go func() {
		c.rotateSoughtNodeID(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("rotateSoughtNodeID did not return with already-cancelled context")
	}
}

func TestCrawler_Start_StopsOnStoppedChannel(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	ctx := context.Background()

	done := make(chan struct{})

	go func() {
		c.start(ctx)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)

	close(c.stopped)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("start did not return after stopped channel was closed")
	}
}

func TestCrawler_Start_StopsOnCancelledContext(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.start(ctx)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("start did not return after context cancellation")
	}
}

func TestCrawler_GetNodesForFindNode_Cancellation(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})

	go func() {
		c.getNodesForFindNode(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("getNodesForFindNode did not return with cancelled context")
	}
}

func TestCrawler_GetNodesForFindNode_EmptyTable(t *testing.T) {
	t.Parallel()

	kt := ktable_mocks.NewTable(t)
	kt.On("GetOldestNodes", mock.Anything, mock.Anything).Return([]ktable.Node{})

	c := &crawler{
		kTable:           kt,
		nodesForFindNode: concurrency.NewBufferedConcurrentChannel[ktable.Node](10, 1),
		logger:           zap.NewNop().Sugar(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.getNodesForFindNode(ctx)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("getNodesForFindNode did not return")
	}
}

func TestCrawler_GetOldNodes_Cancellation(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	c.getOldestNodesInterval = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.getOldNodes(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("getOldNodes did not return")
	}
}

func TestCrawler_GetNodesForSampleInfoHashes_Cancellation(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})

	go func() {
		c.getNodesForSampleInfoHashes(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("getNodesForSampleInfoHashes did not return with cancelled context")
	}
}

func TestCrawler_GetNodesForSampleInfoHashes_EmptyTable(t *testing.T) {
	t.Parallel()

	kt := ktable_mocks.NewTable(t)
	kt.On("GetNodesForSampleInfoHashes", mock.Anything).Return([]ktable.Node{})

	c := &crawler{
		kTable:                   kt,
		nodesForSampleInfoHashes: concurrency.NewBufferedConcurrentChannel[ktable.Node](10, 1),
		logger:                   zap.NewNop().Sugar(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.getNodesForSampleInfoHashes(ctx)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("getNodesForSampleInfoHashes did not return")
	}
}

func TestCrawler_RunDiscoveredNodes_Cancellation(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})

	go func() {
		c.runDiscoveredNodes(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runDiscoveredNodes did not return with cancelled context")
	}
}

func TestCrawler_RunDiscoveredNodes_EmptyBatch(t *testing.T) {
	t.Parallel()

	kt := ktable_mocks.NewTable(t)
	kt.On("FilterKnownAddrs", mock.Anything).Return([]netip.Addr{}).Maybe()

	c := &crawler{
		kTable:                   kt,
		discoveredNodes:          concurrency.NewBatchingChannel[ktable.Node](10, 10, time.Hour),
		nodesForPing:             concurrency.NewBufferedConcurrentChannel[ktable.Node](10, 1),
		nodesForFindNode:         concurrency.NewBufferedConcurrentChannel[ktable.Node](10, 1),
		nodesForSampleInfoHashes: concurrency.NewBufferedConcurrentChannel[ktable.Node](10, 1),
		logger:                   zap.NewNop().Sugar(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.runDiscoveredNodes(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runDiscoveredNodes did not return")
	}
}

func TestCrawler_ReseedBootstrapNodes_Cancellation(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.reseedBootstrapNodes(ctx)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reseedBootstrapNodes did not return")
	}
}

func TestCrawler_ReseedBootstrapNodes_WithBootstrapAddrs(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	cfg := *c.config.Load()
	cfg.bootstrapNodes = []string{"127.0.0.1:6881"}
	c.config.Store(&cfg)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.reseedBootstrapNodes(ctx)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reseedBootstrapNodes did not return after context cancellation")
	}
}

func TestCrawler_RunPing_Cancellation(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})

	go func() {
		c.runPing(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runPing did not return with cancelled context")
	}
}

func TestCrawler_RunPing_WithNodeAndError(t *testing.T) {
	t.Parallel()

	kt := ktable_mocks.NewTable(t)
	kt.On("BatchCommand", mock.Anything).Return()

	c := &crawler{
		kTable:       kt,
		client:       &mockClient{pingErr: errors.New("ping failed")},
		nodesForPing: concurrency.NewBufferedConcurrentChannel[ktable.Node](10, 1),
		runtime: &Runtime{
			SeenConnectedPeers:  newSeenPeersBloom(),
			SeenDiscoveredPeers: newSeenPeersBloom(),
		},
		logger: zap.NewNop().Sugar(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.runPing(ctx)
		close(done)
	}()

	c.nodesForPing.In() <- ktable.NewNode(
		testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		netip.MustParseAddrPort("1.2.3.4:6881"),
	)

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runPing did not return")
	}
}

func TestCrawler_RunFindNode_Cancellation(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})

	go func() {
		c.runFindNode(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runFindNode did not return with cancelled context")
	}
}

func TestCrawler_RunFindNode_WithNodeAndError(t *testing.T) {
	t.Parallel()

	kt := ktable_mocks.NewTable(t)
	kt.On("BatchCommand", mock.Anything).Return()

	c := &crawler{
		kTable:           kt,
		client:           &mockClient{findNodeErr: errors.New("find_node failed")},
		nodesForFindNode: concurrency.NewBufferedConcurrentChannel[ktable.Node](10, 1),
		soughtNodeID:     &concurrency.AtomicValue[protocol.ID]{},
		discoveredNodes:  concurrency.NewBatchingChannel[ktable.Node](10, 10, time.Hour),
		runtime: &Runtime{
			SeenConnectedPeers:  newSeenPeersBloom(),
			SeenDiscoveredPeers: newSeenPeersBloom(),
			PeersDiscovered:     &concurrency.AtomicValue[uint64]{},
		},
		logger: zap.NewNop().Sugar(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		c.runFindNode(ctx)
		close(done)
	}()

	c.nodesForFindNode.In() <- ktable.NewNode(
		testutil.MustParseID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		netip.MustParseAddrPort("2.3.4.5:6881"),
	)

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runFindNode did not return")
	}
}

func TestCrawler_RunSampleInfoHashes_Cancellation(t *testing.T) {
	t.Parallel()

	c := newMinimalCrawler(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})

	go func() {
		c.runSampleInfoHashes(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runSampleInfoHashes did not return with cancelled context")
	}
}
