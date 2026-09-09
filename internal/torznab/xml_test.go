package torznab

import (
	"encoding/xml"
	"net/url"
	"strings"
	"testing"
	"time"

	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeedFor(t *testing.T) {
	t.Parallel()

	baseURL, err := url.Parse("http://localhost:3333/torznab")
	require.NoError(t, err)

	seeders := int32(42)
	leechers := int32(7)
	ct := "movie"
	cs := "imdb"
	cid := "tt0133093"
	created := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)

	rows := []dbsearch.TorrentSearchRow{
		{
			InfoHash:      "abcdef1234567890abcdef1234567890abcdef12",
			TorrentName:   "The Matrix 1999 1080p BluRay x264",
			Size:          1234567890,
			Seeders:       &seeders,
			Leechers:      &leechers,
			ContentType:   &ct,
			ContentSource: &cs,
			ContentID:     &cid,
			CreatedAt:     created,
		},
	}

	body, err := feedFor(baseURL, rows)
	require.NoError(t, err)

	doc := string(body)

	assert.Contains(t, doc, `<rss version="2.0"`)
	assert.Contains(t, doc, `xmlns:torznab="http://torznab.com/schemas/2015/feed"`)
	assert.Contains(t, doc, "<title>The Matrix 1999 1080p BluRay x264</title>")
	assert.Contains(t, doc, `<guid isPermaLink="false">abcdef1234567890abcdef1234567890abcdef12</guid>`)
	assert.Contains(t, doc, "<category>Movies</category>")
	assert.Contains(t, doc, "<size>1234567890</size>")
	assert.Contains(t, doc, `name="infohash" value="abcdef1234567890abcdef1234567890abcdef12"`)
	assert.Contains(t, doc, `name="seeders" value="42"`)
	assert.Contains(t, doc, `name="leechers" value="7"`)
	assert.Contains(t, doc, `name="peers" value="49"`)
	assert.Contains(t, doc, `name="size" value="1234567890"`)
	assert.Contains(t, doc, `name="category" value="2000"`)
	assert.Contains(t, doc, `name="imdbid" value="tt0133093"`)
	assert.Contains(t, doc, `magnet:?`)
	assert.Contains(t, doc, "xt=urn%3Abtih%3Aabcdef1234567890abcdef1234567890abcdef12")
	assert.Contains(t, doc, "http://localhost:3333/api/torrents/abcdef1234567890abcdef1234567890abcdef12/download")

	enclosureURL := "http://localhost:3333/api/torrents/abcdef1234567890abcdef1234567890abcdef12/download"
	assert.Contains(t, doc, `<enclosure url="`+enclosureURL+`" type="application/x-bittorrent" length="0"`)

	var parsed rss
	require.NoError(t, xml.Unmarshal(body, &parsed))
	require.Len(t, parsed.Channel.Items, 1)
}

func TestFeedForEmpty(t *testing.T) {
	t.Parallel()

	baseURL, err := url.Parse("http://localhost:3333")
	require.NoError(t, err)

	body, err := feedFor(baseURL, nil)
	require.NoError(t, err)

	var parsed rss
	require.NoError(t, xml.Unmarshal(body, &parsed))
	assert.Empty(t, parsed.Channel.Items)
}

func TestFeedForUncategorisedTorrent(t *testing.T) {
	t.Parallel()

	baseURL, err := url.Parse("http://localhost:3333")
	require.NoError(t, err)

	rows := []dbsearch.TorrentSearchRow{
		{
			InfoHash:    "abcdef1234567890abcdef1234567890abcdef12",
			TorrentName: "Some.App.v1.0",
			Size:        100,
			CreatedAt:   time.Now(),
		},
	}

	body, err := feedFor(baseURL, rows)
	require.NoError(t, err)
	assert.Contains(t, string(body), `name="category" value="8000"`)
	assert.Contains(t, string(body), "<category>Other</category>")
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()

	body := errorResponse(errCodeParameterMissing, "missing parameter: t")
	doc := string(body)
	assert.Contains(t, doc, `<error code="100" description="missing parameter: t"`)

	// XML must parse as the Newznab error shape
	var parsed struct {
		XMLName     xml.Name `xml:"error"`
		Code        int      `xml:"code,attr"`
		Description string   `xml:"description,attr"`
	}
	require.NoError(t, xml.Unmarshal(body, &parsed))
	assert.Equal(t, 100, parsed.Code)
	assert.Equal(t, "missing parameter: t", parsed.Description)
}

func TestMagnetURI(t *testing.T) {
	t.Parallel()

	r := searchResult{
		InfoHash: "abcdef1234567890abcdef1234567890abcdef12",
		Name:     "The Matrix",
		Size:     1000,
	}

	uri := magnetURI(r)
	assert.True(t, strings.HasPrefix(uri, "magnet:?"))
	assert.Contains(t, uri, "xt=urn%3Abtih%3Aabcdef1234567890abcdef1234567890abcdef12")
	assert.Contains(t, uri, "dn=The+Matrix")
	assert.Contains(t, uri, "xl=1000")
}

func TestDownloadURLFor(t *testing.T) {
	t.Parallel()

	baseURL, err := url.Parse("http://localhost:3333/torznab")
	require.NoError(t, err)

	u, err := downloadURLFor(baseURL, "abcdef1234567890abcdef1234567890abcdef12")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:3333/api/torrents/abcdef1234567890abcdef1234567890abcdef12/download", u)

	_, err = downloadURLFor(baseURL, "")
	require.Error(t, err)
}
