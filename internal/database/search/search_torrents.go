package dbsearch

import (
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/model"
	search "github.com/hexsans/hexmagnet/internal/search"
)

func (s *pgSearch) TorrentsWithMissingInfoHashes(
	ctx context.Context,
	params search.TorrentsWithMissingInfoHashesParams,
) (search.TorrentsWithMissingInfoHashesResult, error) {
	rawTorrents, err := s.q.ListTorrentsByInfoHashes(ctx, params.InfoHashes)
	if err != nil {
		return search.TorrentsWithMissingInfoHashesResult{}, fmt.Errorf("query torrents: %w", err)
	}

	found := make(map[string]model.Torrent, len(rawTorrents))
	for _, rt := range rawTorrents {
		t := db.TorrentToModel(rt)

		files, err := s.q.ListTorrentFiles(ctx, rt.InfoHash)
		if err != nil {
			return search.TorrentsWithMissingInfoHashesResult{}, fmt.Errorf("list files for %s: %w", rt.InfoHash, err)
		}

		for i := range files {
			t.Files = append(t.Files, db.TorrentFileToModel(files[i]))
		}

		found[rt.InfoHash] = t
	}

	var (
		torrents []model.Torrent
		missing  []string
	)

	for _, h := range params.InfoHashes {
		if t, ok := found[h]; ok {
			torrents = append(torrents, t)
		} else {
			missing = append(missing, h)
		}
	}

	return search.TorrentsWithMissingInfoHashesResult{
		Torrents:          torrents,
		MissingInfoHashes: missing,
	}, nil
}
