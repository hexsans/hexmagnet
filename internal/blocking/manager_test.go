package blocking

import (
	"context"
	"testing"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockQueries struct {
	blocked map[string]string
}

func (q *mockQueries) GetBlockedInfoHashes(_ context.Context, hashes []string) ([]string, error) {
	var result []string

	for _, h := range hashes {
		if _, ok := q.blocked[h]; ok {
			result = append(result, h)
		}
	}

	return result, nil
}

func (q *mockQueries) UpsertBlockedInfoHash(_ context.Context, arg db.UpsertBlockedInfoHashParams) error {
	q.blocked[arg.InfoHash] = arg.Reason
	return nil
}

func (q *mockQueries) DeleteBlockedInfoHash(_ context.Context, infoHash string) error {
	delete(q.blocked, infoHash)
	return nil
}

func (q *mockQueries) ListBlockedInfoHashes(_ context.Context) ([]db.BlockedInfoHash, error) {
	result := make([]db.BlockedInfoHash, 0, len(q.blocked))
	for infoHash, reason := range q.blocked {
		result = append(result, db.BlockedInfoHash{InfoHash: infoHash, Reason: reason})
	}

	return result, nil
}

func newTestManager() *manager {
	return &manager{
		q:      &mockQueries{blocked: make(map[string]string)},
		logger: zap.NewNop().Sugar(),
	}
}

func idFromBytes(b []byte) protocol.ID {
	var id protocol.ID
	copy(id[:], b)

	return id
}

func TestFilter_EmptyInput(t *testing.T) {
	t.Parallel()

	m := newTestManager()
	ctx := context.Background()

	result, err := m.Filter(ctx, []protocol.ID{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestFilter_AllAllowed(t *testing.T) {
	t.Parallel()

	m := newTestManager()
	ctx := context.Background()

	hashes := []protocol.ID{
		idFromBytes([]byte("aaaaaaaaaaaaaaa1")),
		idFromBytes([]byte("aaaaaaaaaaaaaaa2")),
	}

	result, err := m.Filter(ctx, hashes)
	require.NoError(t, err)
	assert.Equal(t, hashes, result)
}

func TestFilter_SomeBlocked(t *testing.T) {
	t.Parallel()

	m := newTestManager()
	ctx := context.Background()

	blocked := idFromBytes([]byte("bbbbbbbbbbbbbbb1"))
	allowed := idFromBytes([]byte("ccccccccccccccc1"))

	err := m.Block(ctx, blocked, "test reason")
	require.NoError(t, err)

	result, err := m.Filter(ctx, []protocol.ID{blocked, allowed})
	require.NoError(t, err)
	assert.Equal(t, []protocol.ID{allowed}, result)
}

func TestFilter_AllBlocked(t *testing.T) {
	t.Parallel()

	m := newTestManager()
	ctx := context.Background()

	h1 := idFromBytes([]byte("ddddddddddddddd1"))
	h2 := idFromBytes([]byte("ddddddddddddddd2"))

	_ = m.Block(ctx, h1, "r1")
	_ = m.Block(ctx, h2, "r2")

	result, err := m.Filter(ctx, []protocol.ID{h1, h2})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestBlockAndUnblock(t *testing.T) {
	t.Parallel()

	m := newTestManager()
	ctx := context.Background()

	hash := idFromBytes([]byte("eeeeeeeeeeeeeee1"))
	reason := "name too short"

	err := m.Block(ctx, hash, reason)
	require.NoError(t, err)

	list, err := m.ListBlocked(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, reason, list[0].Reason)
	assert.Equal(t, db.FromProtocolID(hash), list[0].InfoHash)

	err = m.Unblock(ctx, hash)
	require.NoError(t, err)

	list, err = m.ListBlocked(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestBlock_DuplicateUpdatesReason(t *testing.T) {
	t.Parallel()

	m := newTestManager()
	ctx := context.Background()

	hash := idFromBytes([]byte("fffffffffffffff1"))

	_ = m.Block(ctx, hash, "first reason")
	_ = m.Block(ctx, hash, "updated reason")

	list, _ := m.ListBlocked(ctx)
	assert.Len(t, list, 1)
	assert.Equal(t, "updated reason", list[0].Reason)
}
