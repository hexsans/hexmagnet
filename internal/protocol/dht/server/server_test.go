package server

import (
	"context"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/anacrolix/torrent/bencode"
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type mockSocket struct {
	mu       sync.Mutex
	openErr  error
	closeErr error
	opened   bool
	sendHook func()
	sent     []mockSent
	recvCh   chan mockRecv
}

type mockSent struct {
	addr netip.AddrPort
	data []byte
}

type mockRecv struct {
	n    int
	addr netip.AddrPort
	data []byte
	err  error
}

func newMockSocket() *mockSocket {
	return &mockSocket{
		recvCh: make(chan mockRecv, 100),
	}
}

func (s *mockSocket) WriteRecv(data []byte, addr netip.AddrPort) {
	s.recvCh <- mockRecv{n: len(data), addr: addr, data: append([]byte(nil), data...)}
}

func (s *mockSocket) CloseRecv() {
	close(s.recvCh)
}

func (s *mockSocket) Open(_ netip.AddrPort) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.opened = true

	return s.openErr
}

func (s *mockSocket) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.closeErr
}

func (s *mockSocket) Send(addr netip.AddrPort, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sent = append(s.sent, mockSent{addr: addr, data: append([]byte(nil), data...)})
	if s.sendHook != nil {
		s.sendHook()
	}

	return nil
}

func (s *mockSocket) Receive(buf []byte) (int, netip.AddrPort, error) {
	recv, ok := <-s.recvCh
	if !ok {
		return 0, netip.AddrPort{}, nil
	}

	n := copy(buf, recv.data)

	return n, recv.addr, recv.err
}

type mockIDIssuer struct {
	mu    sync.Mutex
	count int
}

func (i *mockIDIssuer) Issue() string {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.count++

	return string(rune(i.count))
}

type mockServer struct {
	queryFn func(ctx context.Context, addr netip.AddrPort, q string, args dht.MsgArgs) (dht.RecvMsg, error)
}

func (mockServer) start() error { return nil }

func (mockServer) stop() {}

func (s mockServer) Query(ctx context.Context, addr netip.AddrPort, q string, args dht.MsgArgs) (dht.RecvMsg, error) {
	if s.queryFn != nil {
		return s.queryFn(ctx, addr, q, args)
	}

	return dht.RecvMsg{}, nil
}

func TestVariantIDIssuer_Issue(t *testing.T) {
	t.Parallel()

	issuer := &variantIDIssuer{}
	seen := make(map[string]bool)

	for range 1000 {
		id := issuer.Issue()
		assert.False(t, seen[id], "duplicate ID: %s", id)
		seen[id] = true
	}
}

func TestVariantIDIssuer_Issue_Ordered(t *testing.T) {
	t.Parallel()

	issuer := &variantIDIssuer{}
	first := issuer.Issue()
	second := issuer.Issue()
	assert.NotEqual(t, first, second)
}

func TestServer_Send(t *testing.T) {
	t.Parallel()

	sock := newMockSocket()
	srv := &server{
		socket:  sock,
		queries: make(map[string]chan dht.RecvMsg),
		logger:  zap.NewNop().Sugar(),
	}

	addr := netip.MustParseAddrPort("1.2.3.4:6881")
	msg := dht.Msg{
		T: "test",
		Y: dht.YQuery,
		Q: "ping",
	}

	err := srv.send(addr, msg)
	require.NoError(t, err)

	sock.mu.Lock()
	require.Len(t, sock.sent, 1)
	sentData := sock.sent[0].data
	sock.mu.Unlock()

	var decoded dht.Msg

	err = bencode.Unmarshal(sentData, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "ping", decoded.Q)
	assert.Equal(t, dht.YQuery, decoded.Y)
}

func TestServer_Query_Success(t *testing.T) {
	t.Parallel()

	sock := newMockSocket()
	issuer := &mockIDIssuer{}
	addr := netip.MustParseAddrPort("1.2.3.4:6881")

	responseMsg := dht.Msg{
		T: "\x01",
		Y: dht.YResponse,
		R: &dht.Return{},
	}
	respData, _ := bencode.Marshal(responseMsg)
	sock.WriteRecv(respData, addr)

	srv := &server{
		stopped:      make(chan struct{}),
		socket:       sock,
		idIssuer:     issuer,
		localAddr:    netip.MustParseAddrPort("0.0.0.0:0"),
		queries:      make(map[string]chan dht.RecvMsg),
		queryTimeout: time.Second,
		logger:       zap.NewNop().Sugar(),
	}

	err := srv.start()
	require.NoError(t, err)

	defer srv.stop()

	res, err := srv.Query(context.Background(), addr, "ping", dht.MsgArgs{})
	require.NoError(t, err)
	assert.Equal(t, dht.YResponse, res.Msg.Y)
}

func TestServer_Query_Timeout(t *testing.T) {
	t.Parallel()

	sock := newMockSocket()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	srv := &server{
		socket:       sock,
		idIssuer:     &mockIDIssuer{},
		queries:      make(map[string]chan dht.RecvMsg),
		queryTimeout: 10 * time.Millisecond,
		logger:       zap.NewNop().Sugar(),
	}

	_, err := srv.Query(ctx, netip.MustParseAddrPort("1.2.3.4:6881"), "ping", dht.MsgArgs{})
	assert.Error(t, err)
}

func TestServer_HandleResponse(t *testing.T) {
	t.Parallel()

	ch := make(chan dht.RecvMsg, 1)

	srv := &server{
		queries: map[string]chan dht.RecvMsg{
			"tx1": ch,
		},
		logger: zap.NewNop().Sugar(),
	}

	srv.handleResponse(dht.RecvMsg{
		Msg: dht.Msg{
			T: "tx1",
			Y: dht.YResponse,
			R: &dht.Return{},
		},
		From: netip.MustParseAddrPort("1.2.3.4:6881"),
	})

	select {
	case res := <-ch:
		assert.Equal(t, "tx1", res.Msg.T)
	case <-time.After(time.Second):
		t.Fatal("message not dispatched to channel")
	}
}

func TestServer_HandleResponse_NoChannel(t *testing.T) {
	t.Parallel()

	srv := &server{
		queries: make(map[string]chan dht.RecvMsg),
		logger:  zap.NewNop().Sugar(),
	}

	srv.handleResponse(dht.RecvMsg{
		Msg: dht.Msg{T: "unknown"},
	})
}

func TestQueryLimiter_PassesThrough(t *testing.T) {
	t.Parallel()

	var called bool

	inner := mockServer{
		queryFn: func(_ context.Context, _ netip.AddrPort, _ string, _ dht.MsgArgs) (dht.RecvMsg, error) {
			called = true
			return dht.RecvMsg{}, nil
		},
	}

	lim := queryLimiter{
		server:       inner,
		queryLimiter: concurrency.NewKeyedLimiter(rate.Inf, 1, 100, time.Minute),
	}

	_, err := lim.Query(context.Background(), netip.MustParseAddrPort("1.2.3.4:6881"), "ping", dht.MsgArgs{})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestQueryLimiter_Blocks(t *testing.T) {
	t.Parallel()

	lim := queryLimiter{
		server:       mockServer{},
		queryLimiter: concurrency.NewKeyedLimiter(1, 1, 100, time.Minute),
	}

	ctx := context.Background()
	addr := netip.MustParseAddrPort("1.2.3.4:6881")

	_, err := lim.Query(ctx, addr, "ping", dht.MsgArgs{})
	require.NoError(t, err)

	ctx2, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	_, err = lim.Query(ctx2, addr, "ping", dht.MsgArgs{})
	assert.Error(t, err)
}

func TestHealthCollector_TracksTimestamps(t *testing.T) {
	t.Parallel()

	lr := &concurrency.AtomicValue[LastResponses]{}
	inner := mockServer{
		queryFn: func(_ context.Context, _ netip.AddrPort, _ string, _ dht.MsgArgs) (dht.RecvMsg, error) {
			return dht.RecvMsg{}, nil
		},
	}

	hc := healthCollector{
		baseServer:    inner,
		lastResponses: lr,
	}

	err := hc.start()
	require.NoError(t, err)

	_, err = hc.Query(context.Background(), netip.MustParseAddrPort("1.2.3.4:6881"), "ping", dht.MsgArgs{})
	require.NoError(t, err)

	result := lr.Get()
	assert.False(t, result.StartTime.IsZero())
	assert.False(t, result.LastResponse.IsZero())
	assert.False(t, result.LastSuccess.IsZero())
}

func TestHealthCollector_TracksFailure(t *testing.T) {
	t.Parallel()

	lr := &concurrency.AtomicValue[LastResponses]{}
	inner := mockServer{
		queryFn: func(_ context.Context, _ netip.AddrPort, _ string, _ dht.MsgArgs) (dht.RecvMsg, error) {
			return dht.RecvMsg{}, assert.AnError
		},
	}

	hc := healthCollector{
		baseServer:    inner,
		lastResponses: lr,
	}

	_, err := hc.Query(context.Background(), netip.MustParseAddrPort("1.2.3.4:6881"), "ping", dht.MsgArgs{})
	require.Error(t, err)

	result := lr.Get()
	assert.False(t, result.LastResponse.IsZero())
	assert.True(t, result.LastSuccess.IsZero())
}

func TestHealthCollector_StopResets(t *testing.T) {
	t.Parallel()

	lr := &concurrency.AtomicValue[LastResponses]{}
	lr.Set(LastResponses{StartTime: time.Now(), LastResponse: time.Now(), LastSuccess: time.Now()})

	hc := healthCollector{
		baseServer:    mockServer{},
		lastResponses: lr,
	}

	hc.stop()

	result := lr.Get()
	assert.True(t, result.StartTime.IsZero())
}

func TestGlobalRequestRateLimiter_PassesThrough(t *testing.T) {
	t.Parallel()

	var called bool

	inner := mockServer{
		queryFn: func(_ context.Context, _ netip.AddrPort, _ string, _ dht.MsgArgs) (dht.RecvMsg, error) {
			called = true
			return dht.RecvMsg{}, nil
		},
	}

	grrl := globalRequestRateLimiter{
		server:  inner,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	_, err := grrl.Query(context.Background(), netip.MustParseAddrPort("1.2.3.4:6881"), "ping", dht.MsgArgs{})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestGlobalRequestRateLimiter_Blocks(t *testing.T) {
	t.Parallel()

	grrl := globalRequestRateLimiter{
		server:  mockServer{},
		limiter: rate.NewLimiter(1, 1),
	}

	ctx := context.Background()
	addr := netip.MustParseAddrPort("1.2.3.4:6881")

	_, err := grrl.Query(ctx, addr, "ping", dht.MsgArgs{})
	require.NoError(t, err)

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()

	_, err = grrl.Query(ctx2, addr, "ping", dht.MsgArgs{})
	assert.Error(t, err)
}

func TestServer_StartStop(t *testing.T) {
	t.Parallel()

	sock := newMockSocket()
	srv := &server{
		stopped:   make(chan struct{}),
		socket:    sock,
		localAddr: netip.MustParseAddrPort("0.0.0.0:0"),
		queries:   make(map[string]chan dht.RecvMsg),
		logger:    zap.NewNop().Sugar(),
	}

	err := srv.start()
	require.NoError(t, err)
	assert.True(t, sock.opened)

	srv.stop()
}
