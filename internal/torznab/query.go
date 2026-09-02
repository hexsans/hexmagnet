package torznab

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hexsans/hexmagnet/internal/model"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
)

// Torznab query types (the ?t= parameter).
const (
	tCapabilities = "capabilities"
	tSearch       = "search"
	tMovie        = "movie"
	tTvSearch     = "tvsearch"
	tMusic        = "music"
	tBook         = "book"
	tGet          = "get"
	tDetails      = "details"
	tDownload     = "download"
	// tTvLegacy is the legacy ?t=tv alias for tvsearch.
	tTvLegacy = "tv"
)

// query is a parsed Torznab request.
type query struct {
	// T is the operation type (?t=).
	T string
	// Q is the free-text search term.
	Q string
	// ImdbID / TmdbID / TvdbID are external content identifiers.
	ImdbID string
	TmdbID string
	TvdbID string
	// Season / Ep are TV search filters.
	Season string
	Ep     string
	// Artist / Album / Label are music search filters (Lidarr).
	Artist string
	Album  string
	Label  string
	// Title / Author are book search filters (Readarr).
	Title  string
	Author string
	// Cat is the comma-separated Newznab category filter.
	Cat string
	// Limit / Offset control pagination.
	Limit  int
	Offset int
	// ID is the info hash for get/details/download requests.
	ID string
	// APIKey is the client-supplied key.
	APIKey string
}

// parseQuery parses Torznab query parameters.
func parseQuery(values url.Values) (query, error) {
	q := query{
		T:      strings.ToLower(values.Get("t")),
		Q:      values.Get("q"),
		ImdbID: values.Get("imdbid"),
		TmdbID: values.Get("tmdbid"),
		TvdbID: values.Get("tvdbid"),
		Season: values.Get("season"),
		Ep:     values.Get("ep"),
		Artist: values.Get("artist"),
		Album:  values.Get("album"),
		Label:  values.Get("label"),
		Title:  values.Get("title"),
		Author: values.Get("author"),
		Cat:    values.Get("cat"),
		ID:     values.Get("id"),
		APIKey: values.Get("apikey"),
	}

	if q.T == "" {
		return query{}, errMissingParameter("t")
	}

	var err error

	if raw := values.Get("limit"); raw != "" {
		q.Limit, err = strconv.Atoi(raw)
		if err != nil || q.Limit < 1 {
			return query{}, fmt.Errorf("invalid limit %q", raw)
		}
	}

	if raw := values.Get("offset"); raw != "" {
		q.Offset, err = strconv.Atoi(raw)
		if err != nil || q.Offset < 0 {
			return query{}, fmt.Errorf("invalid offset %q", raw)
		}
	}

	if q.ImdbID != "" && !strings.HasPrefix(q.ImdbID, "tt") {
		return query{}, fmt.Errorf("invalid imdbid %q", q.ImdbID)
	}

	return q, nil
}

// toSearchParams translates a parsed Torznab query into the internal search
// parameters used by the search backends.
func (q query) toSearchParams(cfg Config) (dbsearch.TorrentSearchParams, error) {
	params := dbsearch.TorrentSearchParams{
		Limit:      uint(capResults(q.Limit, cfg.MaxResults)),
		Offset:     uint(q.Offset),
		TotalCount: true,
		OrderBy: []dbsearch.TorrentSearchOrder{
			{Field: dbsearch.FieldSeeders, Direction: dbsearch.SortDesc},
		},
	}

	switch q.T {
	case tSearch:
		params.QueryString = q.Q

		types, err := parseCategories(q.Cat)
		if err != nil {
			return dbsearch.TorrentSearchParams{}, err
		}

		params.ContentTypes = contentTypeStrings(types)
	case tMovie:
		params.QueryString = q.Q
		params.ContentTypes = []string{string(model.ContentTypeMovie)}

		ref := q.contentRef(model.ContentTypeMovie)

		if ref != nil {
			params.ContentRefs = []dbsearch.ContentRef{*ref}
		}
	case tTvSearch:
		params.QueryString = q.Q
		params.ContentTypes = []string{string(model.ContentTypeTvShow)}

		ref := q.contentRef(model.ContentTypeTvShow)

		if ref != nil {
			params.ContentRefs = []dbsearch.ContentRef{*ref}
		}

		params.QueryString = q.withSeasonEpisode(params.QueryString)
	case tMusic:
		params.ContentTypes = []string{string(model.ContentTypeMusic)}

		parts := []string{q.Artist, q.Album, q.Label}
		if q.Q != "" {
			parts = append(parts, q.Q)
		}

		params.QueryString = strings.Join(nonEmpty(parts), " ")
	case tBook:
		params.ContentTypes = []string{
			string(model.ContentTypeEbook),
			string(model.ContentTypeComic),
			string(model.ContentTypeAudiobook),
		}

		parts := []string{q.Title, q.Author}
		if q.Q != "" {
			parts = append(parts, q.Q)
		}

		params.QueryString = strings.Join(nonEmpty(parts), " ")
	case tGet, tDetails:
		if q.ID == "" {
			return dbsearch.TorrentSearchParams{}, errMissingParameter("id")
		}

		params.InfoHashes = []string{strings.ToLower(q.ID)}
	default:
		return dbsearch.TorrentSearchParams{}, fmt.Errorf("unsupported query type %q", q.T)
	}

	return params, nil
}

// contentRef builds a content reference from imdbid/tmdbid/tvdbid parameters.
func (q query) contentRef(contentType model.ContentType) *dbsearch.ContentRef {
	switch {
	case q.ImdbID != "":
		return &dbsearch.ContentRef{
			Type:   contentType.String(),
			Source: model.SourceImdb,
			ID:     q.ImdbID,
		}
	case q.TmdbID != "":
		return &dbsearch.ContentRef{
			Type:   contentType.String(),
			Source: model.SourceTmdb,
			ID:     q.TmdbID,
		}
	case q.TvdbID != "":
		return &dbsearch.ContentRef{
			Type:   contentType.String(),
			Source: model.SourceTvdb,
			ID:     q.TvdbID,
		}
	default:
		return nil
	}
}

// withSeasonEpisode appends SxxEyy tokens to the query string so full-text
// search matches torrent names containing the season/episode pattern (e.g.
// "Show.S01E05.720p"). The torrents table does not store parsed episodes, so
// this name-token matching is the primary filter; exact content-id matching
// still applies when tvdbid/tmdbid/imdbid is supplied.
func (q query) withSeasonEpisode(queryString string) string {
	season, seasonErr := strconv.Atoi(q.Season)
	ep, epErr := strconv.Atoi(q.Ep)

	if seasonErr != nil && epErr != nil {
		return queryString
	}

	var token string

	switch {
	case seasonErr == nil && epErr == nil:
		token = fmt.Sprintf("S%02dE%02d", season, ep)
	case seasonErr == nil:
		token = fmt.Sprintf("S%02d", season)
	default:
		token = fmt.Sprintf("E%02d", ep)
	}

	parts := nonEmpty([]string{queryString, token})

	return strings.Join(parts, " ")
}

func contentTypeStrings(types []model.ContentType) []string {
	if len(types) == 0 {
		return nil
	}

	out := make([]string, len(types))
	for i, t := range types {
		out[i] = string(t)
	}

	return out
}

func capResults(requested, maxResults int) int {
	if maxResults <= 0 {
		maxResults = defaultMaxResults
	}

	if requested > 0 && requested < maxResults {
		return requested
	}

	return maxResults
}

func nonEmpty(parts []string) []string {
	var out []string

	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return out
}

type missingParameterError struct {
	param string
}

func (e missingParameterError) Error() string {
	return "missing parameter: " + e.param
}

func errMissingParameter(param string) error {
	return missingParameterError{param: param}
}
