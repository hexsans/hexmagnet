package classifier

import (
	"errors"

	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/tmdb"
)

func (c executionContext) tmdbSearchMovie(title string, year model.Year) (model.Content, error) {
	req := tmdb.SearchMovieRequest{
		Query:        title,
		IncludeAdult: true,
	}
	if !year.IsNil() {
		req.Year = year
	}

	searchResult, searchErr := c.tmdbClient.SearchMovie(c.Context, req)
	if searchErr != nil {
		if c.logger != nil {
			c.logger.Debugw("tmdb search movie failed", "title", title, "error", searchErr)
		}

		return model.Content{}, ErrUnmatched
	}

	bestMatch, ok := levenshteinFindBestMatch[tmdb.SearchMovieResult](
		title,
		searchResult.Results,
		func(item tmdb.SearchMovieResult) []string {
			return []string{item.Title, item.OriginalTitle}
		},
	)

	if !ok {
		if c.logger != nil {
			c.logger.Debugw("tmdb search unmatched", "type", "movie", "query", title)
		}

		return model.Content{}, ErrUnmatched
	}

	if c.logger != nil {
		c.logger.Debugw("tmdb search matched", "type", "movie", "query", title, "matched_id", bestMatch.ID)
	}

	return c.tmdbGetMovieByTMDBID(bestMatch.ID)
}

func (c executionContext) tmdbSearchTVShow(title string, year model.Year) (model.Content, error) {
	req := tmdb.SearchTvRequest{
		Query:        title,
		IncludeAdult: true,
	}
	if !year.IsNil() {
		req.FirstAirDateYear = year
	}

	searchResult, searchErr := c.tmdbClient.SearchTv(c.Context, req)
	if searchErr != nil {
		if c.logger != nil {
			c.logger.Debugw("tmdb search tv show failed", "title", title, "error", searchErr)
		}

		return model.Content{}, ErrUnmatched
	}

	bestMatch, ok := levenshteinFindBestMatch[tmdb.SearchTvResult](
		title,
		searchResult.Results,
		func(item tmdb.SearchTvResult) []string {
			return []string{item.Name, item.OriginalName}
		},
	)

	if !ok {
		if c.logger != nil {
			c.logger.Debugw("tmdb search unmatched", "type", "tv", "query", title)
		}

		return model.Content{}, ErrUnmatched
	}

	if c.logger != nil {
		c.logger.Debugw("tmdb search matched", "type", "tv", "query", title, "matched_id", bestMatch.ID)
	}

	return c.tmdbGetTVShowByTMDBID(bestMatch.ID)
}

func (c executionContext) tmdbGetMovieByTMDBID(id int64) (movie model.Content, err error) {
	d, getDetailsErr := c.tmdbClient.MovieDetails(c.Context, tmdb.MovieDetailsRequest{
		ID: id,
	})
	if getDetailsErr != nil {
		if c.logger != nil && !errors.Is(getDetailsErr, tmdb.ErrNotFound) {
			c.logger.Debugw("tmdb movie details failed", "tmdb_id", id, "error", getDetailsErr)
		}

		if errors.Is(getDetailsErr, tmdb.ErrNotFound) {
			getDetailsErr = ErrUnmatched
		}

		err = getDetailsErr

		return
	}

	return tmdb.MovieDetailsToMovieModel(d)
}

func (c executionContext) tmdbGetTVShowByTMDBID(id int64) (movie model.Content, err error) {
	d, getDetailsErr := c.tmdbClient.TvDetails(c.Context, tmdb.TvDetailsRequest{
		SeriesID:         id,
		AppendToResponse: []string{"external_ids"},
	})
	if getDetailsErr != nil {
		if c.logger != nil && !errors.Is(getDetailsErr, tmdb.ErrNotFound) {
			c.logger.Debugw("tmdb tv show details failed", "tmdb_id", id, "error", getDetailsErr)
		}

		if errors.Is(getDetailsErr, tmdb.ErrNotFound) {
			getDetailsErr = ErrUnmatched
		}

		err = getDetailsErr

		return
	}

	return tmdb.TvShowDetailsToTvShowModel(d)
}
