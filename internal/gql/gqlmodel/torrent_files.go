package gqlmodel

import (
	"context"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/protocol"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"go.uber.org/zap"
)

type TorrentFilesQueryResult struct {
	TotalCount  uint
	HasNextPage bool
	Items       []model.TorrentFile
}

type TorrentFilesQueryInput struct {
	InfoHashes  []protocol.ID
	Limit       model.NullUint
	Page        model.NullUint
	Offset      model.NullUint
	TotalCount  model.NullBool
	HasNextPage model.NullBool
	Cached      model.NullBool
	OrderBy     []gen.TorrentFilesOrderByInput
}

func SearchTorrentFiles(
	ctx context.Context,
	search dbsearch.Search,
	query TorrentFilesQueryInput,
	logger *zap.SugaredLogger,
) (TorrentFilesQueryResult, error) {
	p := ExtractPagination(query.Limit, model.NullUint{}, model.NullUint{}, query.TotalCount, query.HasNextPage)

	if len(query.InfoHashes) > 0 {
		p.Limit = 0
	}

	if query.Page.Valid && query.Page.Uint > 1 {
		p.Offset = (query.Page.Uint - 1) * p.Limit
	} else if query.Offset.Valid {
		p.Offset = query.Offset.Uint
	}

	params := dbsearch.TorrentFilesSearchParams{
		Limit:       p.Limit,
		Offset:      p.Offset,
		TotalCount:  p.TotalCount,
		HasNextPage: p.HasNextPage,
	}

	if len(query.InfoHashes) > 0 {
		params.InfoHash = db.FromProtocolID(query.InfoHashes[0])
	}

	result, err := search.TorrentFiles(ctx, params)
	if err != nil {
		if logger != nil {
			logger.Debugw("torrent files search failed", "info_hash", query.InfoHashes, "error", err)
		}

		return TorrentFilesQueryResult{}, err
	}

	items := make([]model.TorrentFile, len(result.Items))
	for i, tf := range result.Items {
		items[i] = db.TorrentFileToModel(tf)
	}

	return TorrentFilesQueryResult{
		TotalCount:  result.TotalCount,
		HasNextPage: result.HasNextPage,
		Items:       items,
	}, nil
}
