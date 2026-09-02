package torznab

import (
	"net/url"
	"testing"

	"github.com/hexsans/hexmagnet/internal/model"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	t.Parallel()

	t.Run("missing t", func(t *testing.T) {
		t.Parallel()

		_, err := parseQuery(url.Values{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing parameter: t")
	})

	t.Run("basic search", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"search"}, "q": {"dune"}, "limit": {"25"}, "offset": {"10"}})
		require.NoError(t, err)
		assert.Equal(t, "search", q.T)
		assert.Equal(t, "dune", q.Q)
		assert.Equal(t, 25, q.Limit)
		assert.Equal(t, 10, q.Offset)
	})

	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()

		_, err := parseQuery(url.Values{"t": {"search"}, "limit": {"abc"}})
		require.Error(t, err)
	})

	t.Run("tv search with season and episode", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"tvsearch"}, "q": {"the boys"}, "season": {"2"}, "ep": {"5"}})
		require.NoError(t, err)
		assert.Equal(t, "2", q.Season)
		assert.Equal(t, "5", q.Ep)
	})

	t.Run("imdbid validated", func(t *testing.T) {
		t.Parallel()

		_, err := parseQuery(url.Values{"t": {"movie"}, "imdbid": {"12345"}})
		require.Error(t, err)

		q, err := parseQuery(url.Values{"t": {"movie"}, "imdbid": {"tt1234567"}})
		require.NoError(t, err)
		assert.Equal(t, "tt1234567", q.ImdbID)
	})
}

func TestToSearchParams(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()

	t.Run("search maps query string", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"search"}, "q": {"dune 2021"}})
		require.NoError(t, err)

		params, err := q.toSearchParams(cfg)
		require.NoError(t, err)
		assert.Equal(t, "dune 2021", params.QueryString)
		assert.Equal(t, uint(100), params.Limit)
		assert.Equal(t, []dbsearch.TorrentSearchOrder{{Field: dbsearch.FieldSeeders, Direction: dbsearch.SortDesc}}, params.OrderBy)
	})

	t.Run("search with cat filter", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"search"}, "q": {"x"}, "cat": {"2000,5000"}})
		require.NoError(t, err)

		params, err := q.toSearchParams(cfg)
		require.NoError(t, err)
		assert.Equal(t, []string{string(model.ContentTypeMovie), string(model.ContentTypeTvShow)}, params.ContentTypes)
	})

	t.Run("movie with imdbid", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"movie"}, "imdbid": {"tt0133093"}})
		require.NoError(t, err)

		params, err := q.toSearchParams(cfg)
		require.NoError(t, err)
		assert.Equal(t, []string{string(model.ContentTypeMovie)}, params.ContentTypes)
		require.Len(t, params.ContentRefs, 1)
		assert.Equal(t, dbsearch.ContentRef{Type: "movie", Source: "imdb", ID: "tt0133093"}, params.ContentRefs[0])
	})

	t.Run("tv search with season episode appends tokens", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"tvsearch"}, "q": {"the boys"}, "season": {"2"}, "ep": {"5"}})
		require.NoError(t, err)

		params, err := q.toSearchParams(cfg)
		require.NoError(t, err)
		assert.Equal(t, "the boys S02E05", params.QueryString)
		assert.Equal(t, []string{string(model.ContentTypeTvShow)}, params.ContentTypes)
	})

	t.Run("tv search season only", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"tvsearch"}, "q": {"show"}, "season": {"1"}})
		require.NoError(t, err)

		params, err := q.toSearchParams(cfg)
		require.NoError(t, err)
		assert.Equal(t, "show S01", params.QueryString)
	})

	t.Run("music combines artist and album", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"music"}, "artist": {"radiohead"}, "album": {"ok computer"}})
		require.NoError(t, err)

		params, err := q.toSearchParams(cfg)
		require.NoError(t, err)
		assert.Equal(t, "radiohead ok computer", params.QueryString)
		assert.Equal(t, []string{string(model.ContentTypeMusic)}, params.ContentTypes)
	})

	t.Run("book includes ebook comic audiobook", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"book"}, "q": {"dune"}})
		require.NoError(t, err)

		params, err := q.toSearchParams(cfg)
		require.NoError(t, err)
		assert.Equal(
			t,
			[]string{string(model.ContentTypeEbook), string(model.ContentTypeComic), string(model.ContentTypeAudiobook)},
			params.ContentTypes,
		)
	})

	t.Run("get by id", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"get"}, "id": {"ABCDEF1234567890ABCDEF1234567890ABCDEF12"}})
		require.NoError(t, err)

		params, err := q.toSearchParams(cfg)
		require.NoError(t, err)
		assert.Equal(t, []string{"abcdef1234567890abcdef1234567890abcdef12"}, params.InfoHashes)
	})

	t.Run("unsupported type", func(t *testing.T) {
		t.Parallel()

		q, err := parseQuery(url.Values{"t": {"foo"}})
		require.NoError(t, err)

		_, err = q.toSearchParams(cfg)
		require.Error(t, err)
	})

	t.Run("limit capped by config", func(t *testing.T) {
		t.Parallel()

		smallCfg := NewDefaultConfig()
		smallCfg.MaxResults = 10

		q, err := parseQuery(url.Values{"t": {"search"}, "q": {"x"}, "limit": {"50"}})
		require.NoError(t, err)

		params, err := q.toSearchParams(smallCfg)
		require.NoError(t, err)
		assert.Equal(t, uint(10), params.Limit)
	})
}
