package torznab

import (
	"crypto/subtle"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/hexsans/hexmagnet/internal/httpserver"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	Config Config
	Search utils.Lazy[search.Search]
	Logger *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Option  httpserver.Option `group:"http_server_options"`
	Handler *Handler
}

func New(p Params) Result {
	h := &Handler{
		logger: p.Logger.Named("torznab"),
		search: p.Search,
	}
	h.applyConfig(p.Config)

	return Result{
		Option:  h,
		Handler: h,
	}
}

// Handler serves the Torznab API. Routes are dispatched from a NoRoute
// fallback so runtime config updates (including the mount path) take effect
// immediately, without a server restart.
type Handler struct {
	logger *zap.SugaredLogger
	search utils.Lazy[search.Search]
	config atomic.Pointer[Config]
	// allowedTypes is the set of content types searchable under the
	// configured category allow-list; nil means unrestricted.
	allowedTypes atomic.Pointer[allowedTypes]
}

// allowedTypes holds the compiled cfg.Categories allow-list.
type allowedTypes map[model.ContentType]struct{}

// applyConfig stores cfg and (re)compiles the category allow-list.
func (h *Handler) applyConfig(cfg Config) {
	h.config.Store(&cfg)
	h.allowedTypes.Store(compileAllowedTypes(cfg.Categories))
}

// compileAllowedTypes turns the configured Newznab category IDs into a set of
// content types. A nil result means no restriction ("*" or empty config).
func compileAllowedTypes(categories []string) *allowedTypes {
	for _, c := range categories {
		if c == "*" {
			return nil
		}
	}

	if len(categories) == 0 {
		return nil
	}

	set := allowedTypes{}

	for _, c := range categories {
		parsed, err := strconv.Atoi(c)
		if err != nil {
			continue
		}

		for _, ct := range contentTypesForCategory(parsed) {
			set[ct] = struct{}{}
		}
	}

	return &set
}

// Update replaces the runtime config. It is safe for concurrent use and all
// settings (including the mount path) take effect immediately.
func (h *Handler) Update(cfg Config) {
	h.applyConfig(cfg)
}

func (*Handler) Key() string {
	return "torznab"
}

func (h *Handler) Apply(e *gin.Engine) error {
	// Dispatch happens in a global middleware: middleware handlers are
	// additive (unlike NoRoute, which replaces previously registered
	// fallbacks) and run before routing, so runtime config updates —
	// including the mount path — take effect without a restart.
	e.Use(func(c *gin.Context) {
		if h.matchesMount(c.Request.URL.Path) {
			h.handle(c)
			c.Abort()

			return
		}

		c.Next()
	})

	return nil
}

// matchesMount reports whether the request path falls under the currently
// configured API path ({path}/api).
func (h *Handler) matchesMount(reqPath string) bool {
	path := defaultPath
	if cfg := h.config.Load(); cfg != nil && strings.TrimRight(cfg.Path, "/") != "" {
		path = strings.TrimRight(cfg.Path, "/")
	}

	apiPath := path + "/api"

	return reqPath == apiPath || strings.HasPrefix(reqPath, apiPath+"/")
}

func (h *Handler) handle(c *gin.Context) {
	cfg := h.config.Load()
	if cfg == nil || !cfg.Enabled {
		writeXML(c, http.StatusNotFound, errorResponse(errCodeNoSuchFunction, "Torznab API is disabled"))
		return
	}

	q, err := parseQuery(c.Request.URL.Query())
	if err != nil {
		writeXML(c, http.StatusBadRequest, errorResponse(errCodeParameterMissing, err.Error()))
		return
	}

	if cfg.APIKey != "" && !constantTimeEquals(q.APIKey, cfg.APIKey) {
		writeXML(c, http.StatusUnauthorized, errorResponse(errCodeUnauthorized, "Invalid API key"))
		return
	}

	switch q.T {
	case tCapabilities:
		writeXML(c, http.StatusOK, capabilitiesFor(*cfg))
	case tDownload:
		h.handleDownload(c, q)
	case tSearch, tMovie, tTvSearch, tTvLegacy, tMusic, tBook, tGet, tDetails:
		h.handleSearch(c, *cfg, q)
	default:
		writeXML(c, http.StatusBadRequest, errorResponse(errCodeNoSuchFunction, "Unknown query type: "+q.T))
	}
}

// constantTimeEquals compares two strings in constant time.
func constantTimeEquals(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (h *Handler) handleDownload(c *gin.Context, q query) {
	if q.ID == "" {
		writeXML(c, http.StatusBadRequest, errorResponse(errCodeParameterMissing, "missing id parameter"))
		return
	}

	baseURL := h.requestBaseURL(c)

	downloadURL, err := downloadURLFor(baseURL, strings.ToLower(q.ID))
	if err != nil {
		writeXML(c, http.StatusBadRequest, errorResponse(errCodeParameterInvalid, err.Error()))
		return
	}

	c.Redirect(http.StatusFound, downloadURL)
}

func (h *Handler) handleSearch(c *gin.Context, cfg Config, q query) {
	params, err := q.toSearchParams(cfg)
	if err != nil {
		writeXML(c, http.StatusBadRequest, errorResponse(errCodeParameterInvalid, err.Error()))

		return
	}

	// Enforce the configured category allow-list: requests limited to
	// content types outside the list return an empty feed.
	if restricted := h.allowedTypes.Load(); restricted != nil && len(params.ContentTypes) > 0 {
		permitted := intersectTypes(params.ContentTypes, *restricted)
		if len(permitted) == 0 {
			h.writeEmptyFeed(c)

			return
		}

		params.ContentTypes = permitted
	}

	svc, err := h.search.Get()
	if err != nil {
		h.logger.Errorw("failed to get search service", "error", err)
		writeXML(c, http.StatusInternalServerError, errorResponse(errCodeServerError, "search service unavailable"))

		return
	}

	result, err := svc.TorrentSearch(c.Request.Context(), params)
	if err != nil {
		h.logger.Errorw("torznab search failed", "error", err, "type", q.T, "q", q.Q)
		writeXML(c, http.StatusInternalServerError, errorResponse(errCodeServerError, "search failed"))

		return
	}

	feed, err := feedFor(h.requestBaseURL(c), result.Items)
	if err != nil {
		h.logger.Errorw("failed to build torznab feed", "error", err)
		writeXML(c, http.StatusInternalServerError, errorResponse(errCodeServerError, "feed generation failed"))

		return
	}

	writeXML(c, http.StatusOK, feed)
}

func (h *Handler) writeEmptyFeed(c *gin.Context) {
	feed, err := feedFor(h.requestBaseURL(c), nil)
	if err != nil {
		h.logger.Errorw("failed to build torznab feed", "error", err)
		writeXML(c, http.StatusInternalServerError, errorResponse(errCodeServerError, "feed generation failed"))

		return
	}

	writeXML(c, http.StatusOK, feed)
}

func intersectTypes(types []string, allowed map[model.ContentType]struct{}) []string {
	out := make([]string, 0, len(types))

	for _, t := range types {
		if _, ok := allowed[model.ContentType(t)]; ok {
			out = append(out, t)
		}
	}

	return out
}

func writeXML(c *gin.Context, status int, body []byte) {
	c.Data(status, "application/xml; charset=utf-8", body)
}

// requestBaseURL reconstructs the public base URL of the API from the request.
// X-Forwarded-* headers are only honored when trust_proxy_headers is enabled.
func (h *Handler) requestBaseURL(c *gin.Context) *url.URL {
	scheme := "http"

	if h.trustProxy() {
		if forwarded := c.GetHeader("X-Forwarded-Proto"); forwarded != "" {
			scheme = strings.TrimSpace(strings.Split(forwarded, ",")[0])
		} else if c.Request.TLS != nil {
			scheme = "https"
		}
	} else if c.Request.TLS != nil {
		scheme = "https"
	}

	host := c.Request.Host

	if h.trustProxy() {
		if forwarded := c.GetHeader("X-Forwarded-Host"); forwarded != "" {
			host = strings.TrimSpace(strings.Split(forwarded, ",")[0])
		}
	}

	return &url.URL{Scheme: scheme, Host: host}
}

func (h *Handler) trustProxy() bool {
	cfg := h.config.Load()

	return cfg == nil || cfg.TrustProxyHeaders
}
