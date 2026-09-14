package indexer

import (
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/protocol"
)

func loadTorrent(ctx context.Context, q *db.Queries, infoHash protocol.ID) (*model.Torrent, error) {
	raw, err := q.GetTorrent(ctx, db.FromProtocolID(infoHash))
	if err != nil {
		return nil, fmt.Errorf("get torrent for %x: %w", infoHash, err)
	}

	t, err := db.TorrentWithFiles(ctx, q, raw.Torrent, raw.Seeders, raw.Leechers)
	if err != nil {
		return nil, err
	}

	return &t, nil
}
