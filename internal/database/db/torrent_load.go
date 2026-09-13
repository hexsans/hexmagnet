package db

import (
	"context"

	"github.com/hexsans/hexmagnet/internal/model"
)

// TorrentWithFiles maps a raw torrent row and appends its file list. It is the
// single place that turns a torrent plus seeding stats into a model.Torrent
// with Files populated, so seeders/leechers are never dropped.
func TorrentWithFiles(
	ctx context.Context,
	q *Queries,
	raw Torrent,
	seeders, leechers *int32,
) (model.Torrent, error) {
	t := TorrentToModel(raw, seeders, leechers)

	files, err := q.ListTorrentFiles(ctx, raw.InfoHash)
	if err != nil {
		return model.Torrent{}, err
	}

	for i := range files {
		t.Files = append(t.Files, TorrentFileToModel(files[i]))
	}

	return t, nil
}
