package jobcontrol

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyValueStateStore_SaveLoadDelete(t *testing.T) {
	t.Parallel()

	dbtx := newMockDBTX()
	store := NewStateStore(utils.NewLazy(func() (*db.Queries, error) {
		return db.NewQueriesWithDBTX(dbtx), nil
	}))

	ctx := context.Background()

	state, ok, err := store.Load(ctx, KeyReindexState)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, state)

	barrier := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	require.NoError(t, store.Save(ctx, KeyReindexState, State{
		BarrierTime:     barrier,
		CursorCreatedAt: barrier.Add(time.Minute),
		CursorInfoHash:  "abc",
		Total:           100,
		Count:           25,
		Fingerprint:     "fp",
	}))

	loaded, ok, err := store.Load(ctx, KeyReindexState)
	require.NoError(t, err)
	require.True(t, ok)

	assert.True(t, loaded.BarrierTime.Equal(barrier))
	assert.Equal(t, "abc", loaded.CursorInfoHash)
	assert.Equal(t, 100, loaded.Total)
	assert.Equal(t, 25, loaded.Count)
	assert.Equal(t, "fp", loaded.Fingerprint)
	assert.False(t, loaded.UpdatedAt.IsZero())

	require.NoError(t, store.Delete(ctx, KeyReindexState))

	state, ok, err = store.Load(ctx, KeyReindexState)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, state)
}

func TestKeyValueStateStore_LoadInvalidJSON(t *testing.T) {
	t.Parallel()

	dbtx := newMockDBTX()
	dbtx.data[KeyReclassifyState] = []byte("not-json")

	store := NewStateStore(utils.NewLazy(func() (*db.Queries, error) {
		return db.NewQueriesWithDBTX(dbtx), nil
	}))

	state, ok, err := store.Load(context.Background(), KeyReclassifyState)
	require.Error(t, err)
	assert.False(t, ok)
	assert.Empty(t, state)
}

func TestFingerprint_StableAcrossFieldOrder(t *testing.T) {
	t.Parallel()

	type sample struct {
		A string
		B int
	}

	first := Fingerprint(sample{A: "x", B: 1})
	second := Fingerprint(sample{A: "x", B: 1})

	assert.Equal(t, first, second)
	assert.NotEqual(t, first, Fingerprint(sample{A: "x", B: 2}))
}

type mockRow struct {
	scanFn func(dest ...any) error
}

func (m *mockRow) Scan(dest ...any) error {
	return m.scanFn(dest...)
}

type mockDBTX struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMockDBTX() *mockDBTX {
	return &mockDBTX{data: make(map[string][]byte)}
}

func (m *mockDBTX) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, ok := args[0].(string)
	if !ok {
		return pgconn.CommandTag{}, errors.New("unexpected key type")
	}

	if len(args) == 1 {
		delete(m.data, key)

		return pgconn.CommandTag{}, nil
	}

	value, ok := args[1].([]byte)
	if !ok {
		return pgconn.CommandTag{}, errors.New("unexpected value type")
	}

	m.data[key] = value

	return pgconn.CommandTag{}, nil
}

func (*mockDBTX) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query call")
}

func (m *mockDBTX) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, _ := args[0].(string)

	value, ok := m.data[key]
	if !ok {
		return &mockRow{scanFn: func(_ ...any) error {
			return pgx.ErrNoRows
		}}
	}

	return &mockRow{scanFn: func(dest ...any) error {
		*dest[0].(*string) = key
		*dest[1].(*[]byte) = value
		*dest[2].(*pgtype.Timestamptz) = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		*dest[3].(*pgtype.Timestamptz) = pgtype.Timestamptz{Time: time.Now(), Valid: true}

		return nil
	}}
}
