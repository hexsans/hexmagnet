package dbsearch

import (
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/database/db"
	search "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/utils"
)

func (s *pgSearch) TorrentFiles(ctx context.Context, params search.TorrentFilesSearchParams) (search.TorrentFilesResult, error) {
	result := search.TorrentFilesResult{Items: []db.TorrentFile{}}

	limit := utils.ClampInt32(params.Limit)
	offset := utils.ClampInt32(params.Offset)

	if limit > 0 {
		rows, err := s.q.ListTorrentFilesPaginated(ctx, db.ListTorrentFilesPaginatedParams{
			InfoHash: params.InfoHash,
			Limit:    limit,
			Offset:   offset,
		})
		if err != nil {
			return result, fmt.Errorf("query torrent files: %w", err)
		}

		result.Items = rows
	} else {
		rows, err := s.q.ListTorrentFiles(ctx, params.InfoHash)
		if err != nil {
			return result, fmt.Errorf("query torrent files: %w", err)
		}

		result.Items = rows
	}

	if params.TotalCount {
		count, err := s.q.CountTorrentFiles(ctx, params.InfoHash)
		if err != nil {
			return result, fmt.Errorf("count torrent files: %w", err)
		}

		result.TotalCount = uint(count)
	}

	if params.HasNextPage && len(result.Items) > 0 {
		nextExists, err := s.q.TorrentFileExists(ctx, db.TorrentFileExistsParams{
			InfoHash: params.InfoHash,
			Offset:   offset + int32(len(result.Items)),
		})
		if err == nil {
			result.HasNextPage = nextExists
		}
	}

	return result, nil
}
