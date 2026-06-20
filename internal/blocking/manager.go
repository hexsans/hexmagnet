package blocking

import (
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"go.uber.org/zap"
)

type Manager interface {
	Filter(ctx context.Context, hashes []protocol.ID) ([]protocol.ID, error)
	Block(ctx context.Context, hash protocol.ID, reason string) error
	Unblock(ctx context.Context, hash protocol.ID) error
	ListBlocked(ctx context.Context) ([]db.BlockedInfoHash, error)
}

type manager struct {
	q      blockingQueries
	logger *zap.SugaredLogger
}

type blockingQueries interface {
	GetBlockedInfoHashes(ctx context.Context, hashes []string) ([]string, error)
	UpsertBlockedInfoHash(ctx context.Context, arg db.UpsertBlockedInfoHashParams) error
	DeleteBlockedInfoHash(ctx context.Context, infoHash string) error
	ListBlockedInfoHashes(ctx context.Context) ([]db.BlockedInfoHash, error)
}

func (m *manager) Filter(ctx context.Context, hashes []protocol.ID) ([]protocol.ID, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	hexes := make([]string, len(hashes))
	for i, h := range hashes {
		hexes[i] = db.FromProtocolID(h)
	}

	blocked, err := m.q.GetBlockedInfoHashes(ctx, hexes)
	if err != nil {
		return nil, fmt.Errorf("query blocked hashes: %w", err)
	}

	blockedSet := make(map[string]struct{}, len(blocked))
	for _, infoHash := range blocked {
		blockedSet[infoHash] = struct{}{}
	}

	filtered := make([]protocol.ID, 0, len(hashes))
	for _, h := range hashes {
		if _, ok := blockedSet[db.FromProtocolID(h)]; !ok {
			filtered = append(filtered, h)
		}
	}

	return filtered, nil
}

func (m *manager) Block(ctx context.Context, hash protocol.ID, reason string) error {
	return m.q.UpsertBlockedInfoHash(ctx, db.UpsertBlockedInfoHashParams{
		InfoHash: db.FromProtocolID(hash),
		Reason:   reason,
	})
}

func (m *manager) Unblock(ctx context.Context, hash protocol.ID) error {
	return m.q.DeleteBlockedInfoHash(ctx, db.FromProtocolID(hash))
}

func (m *manager) ListBlocked(ctx context.Context) ([]db.BlockedInfoHash, error) {
	return m.q.ListBlockedInfoHashes(ctx)
}
