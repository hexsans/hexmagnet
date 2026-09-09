package torznab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeSearch struct {
	result dbsearch.TorrentSearchResult
	err    error
	called bool
}

func (f *fakeSearch) TorrentSearch(_ context.Context, _ dbsearch.TorrentSearchParams) (dbsearch.TorrentSearchResult, error) {
	f.called = true
	return f.result, f.err
}

func (*fakeSearch) TorrentsWithMissingInfoHashes(
	_ context.Context,
	_ dbsearch.TorrentsWithMissingInfoHashesParams,
) (dbsearch.TorrentsWithMissingInfoHashesResult, error) {
	return dbsearch.TorrentsWithMissingInfoHashesResult{}, nil
}

func (*fakeSearch) TorrentFiles(_ context.Context, _ dbsearch.TorrentFilesSearchParams) (dbsearch.TorrentFilesResult, error) {
	return dbsearch.TorrentFilesResult{}, nil
}

func (*fakeSearch) Close() error { return nil }

func newTestHandler(t *testing.T, cfg Config, svc dbsearch.Search) *Handler {
	t.Helper()

	logger := testutil.NewTestLogger()

	return New(Params{
		Config: cfg,
		Search: utils.NewLazy(func() (dbsearch.Search, error) { return svc, nil }),
		Logger: logger,
	}).Handler
}

func doRequest(t *testing.T, h *Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	g := gin.New()
	require.NoError(t, h.Apply(g))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)

	return w
}

func TestDisabledReturns404(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t, NewDefaultConfig(), &fakeSearch{})
	w := doRequest(t, h, "/torznab/api?t=capabilities")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Torznab API is disabled")
}

func TestCapabilities(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Enabled = true

	h := newTestHandler(t, cfg, &fakeSearch{})
	w := doRequest(t, h, "/torznab/api?t=capabilities")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/xml; charset=utf-8", w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, `<caps`)
	assert.Contains(t, body, `<search available="yes"`)
	assert.Contains(t, body, `<tv-search available="yes"`)
	assert.Contains(t, body, `<movie-search available="yes"`)
	assert.Contains(t, body, `<audio-search available="yes"`)
	assert.Contains(t, body, `<book-search available="yes"`)
	assert.Contains(t, body, `id="2000"`)
	assert.Contains(t, body, `id="5000"`)
}

func TestAPIKeyRequired(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.APIKey = "secret"

	h := newTestHandler(t, cfg, &fakeSearch{})

	w := doRequest(t, h, "/torznab/api?t=capabilities")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid API key")

	w = doRequest(t, h, "/torznab/api?t=capabilities&apikey=secret")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMissingTParameter(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Enabled = true

	h := newTestHandler(t, cfg, &fakeSearch{})
	w := doRequest(t, h, "/torznab/api?q=dune")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing parameter: t")
}

func TestUnknownQueryType(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Enabled = true

	h := newTestHandler(t, cfg, &fakeSearch{})
	w := doRequest(t, h, "/torznab/api?t=foobar")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Unknown query type")
}

func TestSearchReturnsFeed(t *testing.T) {
	t.Parallel()

	seeders := int32(5)
	ct := "music"

	svc := &fakeSearch{
		result: dbsearch.TorrentSearchResult{
			Items: []dbsearch.TorrentSearchRow{
				{
					InfoHash:    "abcdef1234567890abcdef1234567890abcdef12",
					TorrentName: "Artist Album 2024 FLAC",
					Size:        999,
					Seeders:     &seeders,
					ContentType: &ct,
					CreatedAt:   time.Now(),
				},
			},
		},
	}

	cfg := NewDefaultConfig()
	cfg.Enabled = true

	h := newTestHandler(t, cfg, svc)
	w := doRequest(t, h, "/torznab/api?t=music&artist=artist&album=album")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.called)
	assert.Contains(t, w.Body.String(), "<rss version=\"2.0\"")
	assert.Contains(t, w.Body.String(), "Artist Album 2024 FLAC")
}

func TestSearchError(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Enabled = true

	h := newTestHandler(t, cfg, &fakeSearch{err: context.DeadlineExceeded})
	w := doRequest(t, h, "/torznab/api?t=search&q=x")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `code="500"`)
}

func TestDownloadRedirect(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Enabled = true

	h := newTestHandler(t, cfg, &fakeSearch{})
	w := doRequest(t, h, "/torznab/api?t=download&id=ABCDEF1234567890ABCDEF1234567890ABCDEF12")

	assert.Equal(t, http.StatusFound, w.Code)
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "/api/torrents/abcdef1234567890abcdef1234567890abcdef12/download", loc.Path)
}

func TestCustomPath(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Path = "/custom"

	h := newTestHandler(t, cfg, &fakeSearch{})
	w := doRequest(t, h, "/custom/api?t=capabilities")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandlerUpdate(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t, NewDefaultConfig(), &fakeSearch{})

	w := doRequest(t, h, "/torznab/api?t=capabilities")
	assert.Equal(t, http.StatusNotFound, w.Code)

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	h.Update(cfg)

	w = doRequest(t, h, "/torznab/api?t=capabilities")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPathChangeTakesEffectImmediately(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Enabled = true

	h := newTestHandler(t, cfg, &fakeSearch{})

	w := doRequest(t, h, "/torznab/api?t=capabilities")
	assert.Equal(t, http.StatusOK, w.Code)

	cfg.Path = "/newspot"
	h.Update(cfg)

	w = doRequest(t, h, "/newspot/api?t=capabilities")
	assert.Equal(t, http.StatusOK, w.Code)

	// The old mount point must stop working right away.
	w = doRequest(t, h, "/torznab/api?t=capabilities")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCategoriesEnforcedOnSearch(t *testing.T) {
	t.Parallel()

	seeders := int32(5)
	ct := "music"

	svc := &fakeSearch{
		result: dbsearch.TorrentSearchResult{
			Items: []dbsearch.TorrentSearchRow{
				{
					InfoHash:    "abcdef1234567890abcdef1234567890abcdef12",
					TorrentName: "Artist Album 2024 FLAC",
					Size:        999,
					Seeders:     &seeders,
					ContentType: &ct,
					CreatedAt:   time.Now(),
				},
			},
		},
	}

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Categories = []string{"3000"} // music only

	h := newTestHandler(t, cfg, svc)

	// Movie searches are outside the allow-list: empty feed, search not called.
	w := doRequest(t, h, "/torznab/api?t=movie&q=dune")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, svc.called)
	assert.NotContains(t, w.Body.String(), "<item>")

	// Music searches pass through.
	w = doRequest(t, h, "/torznab/api?t=music&artist=artist")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.called)
	assert.Contains(t, w.Body.String(), "Artist Album 2024 FLAC")

	// Lifting the restriction applies immediately.
	cfg.Categories = []string{"*"}
	h.Update(cfg)

	w = doRequest(t, h, "/torznab/api?t=movie&q=dune")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Artist Album 2024 FLAC")
}

func TestTrustProxyHeaders(t *testing.T) {
	t.Parallel()

	target := func(path string) string { return "http://client.example" + path }

	t.Run("disabled ignores forwarded headers", func(t *testing.T) {
		t.Parallel()

		cfg := NewDefaultConfig()
		cfg.Enabled = true
		cfg.TrustProxyHeaders = false

		h := newTestHandler(t, cfg, &fakeSearch{})

		g := gin.New()
		require.NoError(t, h.Apply(g))

		req := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			target("/torznab/api?t=download&id=ABCDEF1234567890ABCDEF1234567890ABCDEF12"),
			nil,
		)
		req.Header.Set("X-Forwarded-Host", "proxy.example")
		req.Header.Set("X-Forwarded-Proto", "https")

		w := httptest.NewRecorder()
		g.ServeHTTP(w, req)

		loc, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "http://client.example", loc.Scheme+"://"+loc.Host)
	})

	t.Run("enabled honors forwarded headers", func(t *testing.T) {
		t.Parallel()

		cfg := NewDefaultConfig()
		cfg.Enabled = true
		cfg.TrustProxyHeaders = true

		h := newTestHandler(t, cfg, &fakeSearch{})

		g := gin.New()
		require.NoError(t, h.Apply(g))

		req := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			target("/torznab/api?t=download&id=ABCDEF1234567890ABCDEF1234567890ABCDEF12"),
			nil,
		)
		req.Header.Set("X-Forwarded-Host", "proxy.example")
		req.Header.Set("X-Forwarded-Proto", "https")

		w := httptest.NewRecorder()
		g.ServeHTTP(w, req)

		loc, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "https://proxy.example", loc.Scheme+"://"+loc.Host)
	})
}

func TestNonTorznabPathStill404(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Enabled = true

	h := newTestHandler(t, cfg, &fakeSearch{})

	w := doRequest(t, h, "/unrelated/path")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestCoexistsWithNoRouteFallback reproduces the webui setup: the webui
// registers its own NoRoute handler AFTER the torznab option. Since gin's
// NoRoute replaces earlier fallbacks, torznab must dispatch through
// middleware to keep working.
func TestCoexistsWithNoRouteFallback(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	cfg.Enabled = true

	h := newTestHandler(t, cfg, &fakeSearch{})

	g := gin.New()
	require.NoError(t, h.Apply(g))

	// webui-style SPA fallback registered after torznab.
	g.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.Data(http.StatusOK, "text/html", []byte("<html>spa</html>"))
			return
		}

		c.Status(http.StatusNotFound)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/torznab/api?t=capabilities", nil)
	g.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "<caps")
	assert.Equal(t, "application/xml; charset=utf-8", w.Header().Get("Content-Type"))

	// Unmatched paths still fall through to the SPA fallback.
	w = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/some/spa/route", nil)
	g.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "spa")
}
