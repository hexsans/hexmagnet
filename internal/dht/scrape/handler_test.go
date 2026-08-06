package scrape

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/dhtcrawler"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/client"
	ktable_mocks "github.com/hexsans/hexmagnet/internal/protocol/dht/ktable/mocks"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockClient struct {
	client.Client
	getPeersScrapeFn func(ctx context.Context, addr netip.AddrPort, infoHash protocol.ID) (client.GetPeersScrapeResult, error)
}

func (m *mockClient) GetPeersScrape(ctx context.Context, addr netip.AddrPort, infoHash protocol.ID) (client.GetPeersScrapeResult, error) {
	return m.getPeersScrapeFn(ctx, addr, infoHash)
}

func TestNew(t *testing.T) {
	t.Parallel()

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &mockClient{}, nil
		}),
		KTable: ktable_mocks.NewTable(t),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	assert.NotNil(t, h)
}

func TestHandleScrape_InvalidHash(t *testing.T) {
	t.Parallel()

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &mockClient{}, nil
		}),
		KTable: ktable_mocks.NewTable(t),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.ScrapeMessage{
		InfoHash: "invalid",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleScrape(context.Background(), msg)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestHandleScrape_EmptyHash(t *testing.T) {
	t.Parallel()

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &mockClient{}, nil
		}),
		KTable: ktable_mocks.NewTable(t),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.ScrapeMessage{
		InfoHash: "",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleScrape(context.Background(), msg)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestHandleScrape_ShortHash(t *testing.T) {
	t.Parallel()

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &mockClient{}, nil
		}),
		KTable: ktable_mocks.NewTable(t),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.ScrapeMessage{
		InfoHash: "abcdef",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleScrape(context.Background(), msg)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestHandleScrape_ClientError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("client unavailable")
	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return nil, expectedErr
		}),
		KTable: ktable_mocks.NewTable(t),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.ScrapeMessage{
		InfoHash: "0123456789abcdef0123456789abcdef01234567",
		Node:     "1.2.3.4:6881",
	}

	_, err := h.HandleScrape(context.Background(), msg)
	require.Error(t, err)
}

func TestHandleScrape_InvalidNode(t *testing.T) {
	t.Parallel()

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &mockClient{}, nil
		}),
		KTable: ktable_mocks.NewTable(t),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.ScrapeMessage{
		InfoHash: "0123456789abcdef0123456789abcdef01234567",
		Node:     "not-a-valid-addr",
	}

	_, err := h.HandleScrape(context.Background(), msg)
	require.Error(t, err)
}

func TestHandleScrape_WithRuntime(t *testing.T) {
	t.Parallel()

	runtime := dhtcrawler.NewRuntime()

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &mockClient{}, nil
		}),
		KTable:  ktable_mocks.NewTable(t),
		Runtime: runtime,
		Logger:  testutil.NewTestLogger(),
	}
	h := New(p)
	assert.NotNil(t, h)
}

func TestHandleScrape_ScrapeNotSupported(t *testing.T) {
	t.Parallel()

	p := Params{
		Client: utils.NewLazy(func() (client.Client, error) {
			return &mockClient{
				getPeersScrapeFn: func(_ context.Context, _ netip.AddrPort, _ protocol.ID) (client.GetPeersScrapeResult, error) {
					return client.GetPeersScrapeResult{}, client.ErrNoScrapeSupport
				},
			}, nil
		}),
		KTable: ktable_mocks.NewTable(t),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.ScrapeMessage{
		InfoHash: "0123456789abcdef0123456789abcdef01234567",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleScrape(context.Background(), msg)
	require.NoError(t, err)
	assert.Nil(t, result)
}
