package persistsources

import (
	"context"
	"errors"
	"testing"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	assert.NotNil(t, h)
}

func TestHandlePersistSources_InvalidHash(t *testing.T) {
	t.Parallel()

	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.ScrapeResultMessage{
		InfoHash: "invalid",
	}

	err := h.HandlePersistSources(context.Background(), msg)
	assert.Error(t, err)
}

func TestHandlePersistSources_EmptyHash(t *testing.T) {
	t.Parallel()

	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.ScrapeResultMessage{
		InfoHash: "",
	}

	err := h.HandlePersistSources(context.Background(), msg)
	assert.Error(t, err)
}

func TestHandlePersistSources_ShortHash(t *testing.T) {
	t.Parallel()

	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, errors.New("not initialized")
		}),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.ScrapeResultMessage{
		InfoHash: "abcdef",
	}

	err := h.HandlePersistSources(context.Background(), msg)
	assert.Error(t, err)
}

func TestHandlePersistSources_ValidHash_QueriesError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("queries unavailable")
	p := Params{
		Queries: utils.NewLazy(func() (*db.Queries, error) {
			return nil, expectedErr
		}),
		Logger: testutil.NewTestLogger(),
	}
	h := New(p)
	msg := dht.ScrapeResultMessage{
		InfoHash: "0123456789abcdef0123456789abcdef01234567",
		Seeders:  5,
		Leechers: 3,
	}

	err := h.HandlePersistSources(context.Background(), msg)
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}
