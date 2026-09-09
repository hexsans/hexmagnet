package webhook

import (
	"time"

	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/testutil"
)

func sampleTorrent() model.Torrent {
	id := testutil.MustParseID("abcdef1234567890abcdef1234567890abcdef12")

	return model.Torrent{
		InfoHash:    id,
		Name:        "Artist - Album (2022) FLAC",
		Size:        999,
		CreatedAt:   time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC),
		FilesCount:  model.NewNullUint(8),
		ContentType: model.NewNullContentType("music"),
		Languages:   model.Languages{model.Language("en"): {}},
		Seeders:     model.NewNullUint(5),
		Leechers:    model.NewNullUint(2),
		Files: []model.TorrentFile{
			{
				InfoHash:  id,
				Index:     0,
				PathParts: []string{"Artist - Album (2022) FLAC", "01 - Track.flac"},
				Extension: model.NewNullString("flac"),
				Size:      100,
			},
			{
				InfoHash:  id,
				Index:     1,
				PathParts: []string{"Artist - Album (2022) FLAC", "cover.jpg"},
				Extension: model.NewNullString("jpg"),
				Size:      50,
			},
		},
	}
}
