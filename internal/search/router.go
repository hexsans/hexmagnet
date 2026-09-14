package search

import "context"

// TorrentSearcher is the query-only backend used for free-text torrent search.
// The Elasticsearch backend implements only this subset; the remaining
// operations are always served by the Postgres backend.
type TorrentSearcher interface {
	TorrentSearch(ctx context.Context, params TorrentSearchParams) (TorrentSearchResult, error)
	Close() error
}

type Router struct {
	es TorrentSearcher
	pg Search
}

func NewRouter(es TorrentSearcher, pg Search) *Router {
	return &Router{es: es, pg: pg}
}

func (r *Router) TorrentSearch(ctx context.Context, params TorrentSearchParams) (TorrentSearchResult, error) {
	if params.QueryString != "" {
		return r.es.TorrentSearch(ctx, params)
	}

	return r.pg.TorrentSearch(ctx, params)
}

func (r *Router) TorrentsWithMissingInfoHashes(
	ctx context.Context,
	params TorrentsWithMissingInfoHashesParams,
) (TorrentsWithMissingInfoHashesResult, error) {
	return r.pg.TorrentsWithMissingInfoHashes(ctx, params)
}

func (r *Router) TorrentFiles(ctx context.Context, params TorrentFilesSearchParams) (TorrentFilesResult, error) {
	return r.pg.TorrentFiles(ctx, params)
}

func (r *Router) Close() error {
	if r.es != nil {
		_ = r.es.Close()
	}

	return r.pg.Close()
}
