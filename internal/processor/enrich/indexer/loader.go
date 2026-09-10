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

	t := db.TorrentToModel(raw.Torrent, raw.Seeders, raw.Leechers)

	files, err := q.ListTorrentFiles(ctx, db.FromProtocolID(infoHash))
	if err != nil {
		return nil, err
	}

	for i := range files {
		t.Files = append(t.Files, db.TorrentFileToModel(files[i]))
	}

	return &t, nil
}
