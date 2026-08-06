package search

import "context"

type Router struct {
	es Search
	pg Search
}

func NewRouter(es, pg Search) *Router {
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
