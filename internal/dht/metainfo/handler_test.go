package metainfo

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/hexsans/hexmagnet/internal/blocking"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/client"
	ktable_mocks "github.com/hexsans/hexmagnet/internal/protocol/dht/ktable/mocks"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopClient struct {
	client.Client
}

type mockClient struct {
	client.Client
	getPeersFn func(ctx context.Context, addr netip.AddrPort, infoHash protocol.ID) (client.GetPeersResult, error)
}

func (m *mockClient) GetPeers(ctx context.Context, addr netip.AddrPort, infoHash protocol.ID) (client.GetPeersResult, error) {
	return m.getPeersFn(ctx, addr, infoHash)
}

type noopBlockingManager struct {
	blocking.Manager
}

func TestNew(t *testing.T) {
	t.Parallel()

	p := Params{
		Client:            nil,
		MetainfoRequester: nil,
		BanningChecker:    nil,
		KTable:            nil,
		BlockingManager:   nil,
		Queries:           nil,
		Logger:            testutil.NewTestLogger(),
	}
	h := New(p)
	assert.NotNil(t, h)
}

func TestHandleGetPeers_InvalidHash(t *testing.T) {
	t.Parallel()

	kt := ktable_mocks.NewTable(t)

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &noopClient{}, nil
		}),
		MetainfoRequester: nil,
		BanningChecker:    nil,
		KTable:            kt,
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &noopBlockingManager{}, nil
		}),
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not used")
		}),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.GetPeersMessage{
		InfoHash: "invalid",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleGetPeers(context.Background(), msg)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestHandleGetPeers_EmptyHash(t *testing.T) {
	t.Parallel()

	kt := ktable_mocks.NewTable(t)

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &noopClient{}, nil
		}),
		MetainfoRequester: nil,
		BanningChecker:    nil,
		KTable:            kt,
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &noopBlockingManager{}, nil
		}),
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not used")
		}),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.GetPeersMessage{
		InfoHash: "",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleGetPeers(context.Background(), msg)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestHandleGetPeers_ShortHash(t *testing.T) {
	t.Parallel()

	kt := ktable_mocks.NewTable(t)

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &noopClient{}, nil
		}),
		MetainfoRequester: nil,
		BanningChecker:    nil,
		KTable:            kt,
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &noopBlockingManager{}, nil
		}),
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not used")
		}),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.GetPeersMessage{
		InfoHash: "abcdef",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleGetPeers(context.Background(), msg)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestHandleGetPeers_NoPeersFound(t *testing.T) {
	t.Parallel()

	kt := ktable_mocks.NewTable(t)

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &mockClient{
				getPeersFn: func(_ context.Context, _ netip.AddrPort, _ protocol.ID) (client.GetPeersResult, error) {
					return client.GetPeersResult{}, nil
				},
			}, nil
		}),
		MetainfoRequester: nil,
		BanningChecker:    nil,
		KTable:            kt,
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &noopBlockingManager{}, nil
		}),
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not used")
		}),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.GetPeersMessage{
		InfoHash: "0123456789abcdef0123456789abcdef01234567",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleGetPeers(context.Background(), msg)
	require.NoError(t, err)
	assert.Nil(t, result)
}
