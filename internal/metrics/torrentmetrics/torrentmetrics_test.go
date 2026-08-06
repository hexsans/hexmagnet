package torrentmetrics

import (
	"context"
	"testing"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	q := db.NewQueries(nil)
	lazyQ := utils.NewLazy(func() (*db.Queries, error) { return q, nil })
	p := Params{Queries: lazyQ}
	result := New(p)
	assert.NotNil(t, result.Client)

	client, err := result.Client.Get()
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNew_QueriesError(t *testing.T) {
	t.Parallel()

	lazyQ := utils.NewLazy(func() (*db.Queries, error) { return nil, assert.AnError })
	p := Params{Queries: lazyQ}
	result := New(p)
	_, err := result.Client.Get()
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestClient_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var (
		c client
		_ Client = c
	)

	_ = c
}

func TestRequest_PanicsWithNilDB(t *testing.T) {
	t.Parallel()

	c := client{q: nil}

	assert.Panics(t, func() {
		_, _ = c.Request(context.Background(), Request{})
	})
}

func TestNew_ReturnsResultWithClient(t *testing.T) {
	t.Parallel()

	p := Params{}
	result := New(p)
	assert.NotNil(t, result.Client)
}
