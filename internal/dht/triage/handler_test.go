package triage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/blocking"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBlockingManager struct {
	blocking.Manager
	filterFn func(ctx context.Context, hashes []protocol.ID) ([]protocol.ID, error)
	blockFn  func(ctx context.Context, hash protocol.ID, reason string) error
}

func (m *mockBlockingManager) Filter(ctx context.Context, hashes []protocol.ID) ([]protocol.ID, error) {
	return m.filterFn(ctx, hashes)
}

func (m *mockBlockingManager) Block(ctx context.Context, hash protocol.ID, reason string) error {
	return m.blockFn(ctx, hash, reason)
}

const triageInfoHash = "0123456789abcdef0123456789abcdef01234567"

const triageSelectQuery = "SELECT t.files_count.*"

func newTestHandlerWithMockPool(mockPool pgxmock.PgxPoolIface) Handler {
	return New(Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return db.NewQueriesWithDBTX(mockPool), nil
		}),
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &mockBlockingManager{
				filterFn: func(_ context.Context, _ []protocol.ID) ([]protocol.ID, error) {
					return []protocol.ID{{0x01}}, nil
				},
			}, nil
		}),
		Config: dht.Config{RescrapeThreshold: 3600},
		Logger: testutil.NewTestLogger(),
	})
}

func newTriageTestMessage() dht.DiscoveredHash {
	return dht.DiscoveredHash{
		InfoHash: triageInfoHash,
		Node:     "1.2.3.4:6881",
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &mockBlockingManager{}, nil
		}),
		Config: dht.Config{RescrapeThreshold: 3600},
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	assert.NotNil(t, h)
}

func TestHandleTriage_InvalidHash(t *testing.T) {
	t.Parallel()

	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &mockBlockingManager{}, nil
		}),
		Config: dht.Config{RescrapeThreshold: 3600},
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.DiscoveredHash{
		InfoHash: "invalid",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleTriage(context.Background(), msg)
	assert.Equal(t, ActionDiscard, result.Action)
	assert.Error(t, err)
}

func TestHandleTriage_EmptyHash(t *testing.T) {
	t.Parallel()

	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &mockBlockingManager{}, nil
		}),
		Config: dht.Config{RescrapeThreshold: 3600},
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.DiscoveredHash{
		InfoHash: "",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleTriage(context.Background(), msg)
	assert.Equal(t, ActionDiscard, result.Action)
	assert.Error(t, err)
}

func TestHandleTriage_ShortHash(t *testing.T) {
	t.Parallel()

	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &mockBlockingManager{}, nil
		}),
		Config: dht.Config{RescrapeThreshold: 3600},
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.DiscoveredHash{
		InfoHash: "abcdef",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleTriage(context.Background(), msg)
	assert.Equal(t, ActionDiscard, result.Action)
	assert.Error(t, err)
}

func TestHandleTriage_BlockingManagerError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("blocking manager unavailable")
	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return nil, expectedErr
		}),
		Config: dht.Config{RescrapeThreshold: 3600},
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.DiscoveredHash{
		InfoHash: "0123456789abcdef0123456789abcdef01234567",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleTriage(context.Background(), msg)
	require.Error(t, err)
	assert.Equal(t, ActionDiscard, result.Action)
}

func TestHandleTriage_FilterError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("filter failed")
	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &mockBlockingManager{
				filterFn: func(_ context.Context, _ []protocol.ID) ([]protocol.ID, error) {
					return nil, expectedErr
				},
			}, nil
		}),
		Config: dht.Config{RescrapeThreshold: 3600},
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.DiscoveredHash{
		InfoHash: "0123456789abcdef0123456789abcdef01234567",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleTriage(context.Background(), msg)
	require.Error(t, err)
	assert.Equal(t, ActionDiscard, result.Action)
}

func TestHandleTriage_FilterDiscardsAll(t *testing.T) {
	t.Parallel()

	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		BlockingManager: utils.NewLazy(func() (blocking.Manager, error) {
			return &mockBlockingManager{
				filterFn: func(_ context.Context, _ []protocol.ID) ([]protocol.ID, error) {
					return []protocol.ID{}, nil
				},
			}, nil
		}),
		Config: dht.Config{RescrapeThreshold: 3600},
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.DiscoveredHash{
		InfoHash: "0123456789abcdef0123456789abcdef01234567",
		Node:     "1.2.3.4:6881",
	}

	result, err := h.HandleTriage(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, ActionDiscard, result.Action)
	assert.Nil(t, result.GetPeersMsg)
	assert.Nil(t, result.ScrapeMsg)
}

func TestHandleTriage_UnknownTorrent(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)

	mockPool.ExpectQuery(triageSelectQuery).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"files_count", "seeders", "leechers", "updated_at"}))

	h := newTestHandlerWithMockPool(mockPool)
	result, err := h.HandleTriage(context.Background(), newTriageTestMessage())
	require.NoError(t, err)
	assert.Equal(t, ActionGetPeers, result.Action)

	require.NotNil(t, result.GetPeersMsg)
	assert.Equal(t, triageInfoHash, result.GetPeersMsg.InfoHash)
	assert.Equal(t, "1.2.3.4:6881", result.GetPeersMsg.Node)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestHandleTriage_NoFiles(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)

	updatedAt := time.Now()

	mockPool.ExpectQuery(triageSelectQuery).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"files_count", "seeders", "leechers", "updated_at"}).
			AddRow(nil, int32(10), int32(4), &updatedAt))

	h := newTestHandlerWithMockPool(mockPool)
	result, err := h.HandleTriage(context.Background(), newTriageTestMessage())
	require.NoError(t, err)
	assert.Equal(t, ActionGetPeers, result.Action)

	require.NotNil(t, result.GetPeersMsg)
	assert.Equal(t, triageInfoHash, result.GetPeersMsg.InfoHash)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestHandleTriage_StaleTorrent(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)

	filesCount := int32(2)
	updatedAt := time.Now().Add(-2 * time.Hour)

	mockPool.ExpectQuery(triageSelectQuery).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"files_count", "seeders", "leechers", "updated_at"}).
			AddRow(&filesCount, int32(0), int32(0), &updatedAt))

	h := newTestHandlerWithMockPool(mockPool)
	result, err := h.HandleTriage(context.Background(), newTriageTestMessage())
	require.NoError(t, err)
	assert.Equal(t, ActionScrape, result.Action)

	require.NotNil(t, result.ScrapeMsg)
	assert.Equal(t, triageInfoHash, result.ScrapeMsg.InfoHash)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestHandleTriage_FreshTorrent(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)

	filesCount := int32(2)
	updatedAt := time.Now()

	mockPool.ExpectQuery(triageSelectQuery).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"files_count", "seeders", "leechers", "updated_at"}).
			AddRow(&filesCount, int32(10), int32(4), &updatedAt))

	h := newTestHandlerWithMockPool(mockPool)
	result, err := h.HandleTriage(context.Background(), newTriageTestMessage())
	require.NoError(t, err)
	assert.Equal(t, ActionDiscard, result.Action)
	assert.Nil(t, result.GetPeersMsg)
	assert.Nil(t, result.ScrapeMsg)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestHandleTriage_QueryError(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)

	mockPool.ExpectQuery(triageSelectQuery).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errors.New("db failure"))

	h := newTestHandlerWithMockPool(mockPool)
	result, err := h.HandleTriage(context.Background(), newTriageTestMessage())
	require.Error(t, err)
	assert.Equal(t, ActionDiscard, result.Action)

	require.NoError(t, mockPool.ExpectationsWereMet())
}
