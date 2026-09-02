package torznab

const (
	defaultPath       = "/torznab"
	defaultMaxResults = 100
)

// Config controls the Torznab API endpoint that Servarr apps (Lidarr, Radarr,
// Sonarr, Readarr) and Prowlarr use to search and download crawled torrents.
type Config struct {
	// Enabled turns the Torznab endpoint on or off.
	Enabled bool `yaml:"enabled"`
	// APIKey optionally requires clients to pass ?apikey= on every request.
	// When empty no key is required.
	APIKey string `yaml:"api_key"`
	// Path is the base path the API is served under; the full endpoint is
	// {path}/api, e.g. http://host:3333/torznab/api
	Path string `yaml:"path"`
	// MaxResults caps how many results a single search query returns.
	MaxResults int `yaml:"max_results"`
	// Categories restricts which Newznab categories are searchable. Each entry
	// is a Newznab category ID (2000 movies, 3000 music, 5000 TV, 7000 books,
	// 8000 other). "*" (default) means all categories.
	Categories []string `yaml:"categories"`
	// TrustProxyHeaders controls whether X-Forwarded-Proto / X-Forwarded-Host
	// headers are honored when building the public base URL used in enclosure
	// and download links. Enable this only when the server is behind a trusted
	// reverse proxy; when disabled, clients cannot spoof the base URL.
	TrustProxyHeaders bool `yaml:"trust_proxy_headers"`
}

func NewDefaultConfig() Config {
	return Config{
		Enabled:           false,
		APIKey:            "",
		Path:              defaultPath,
		MaxResults:        defaultMaxResults,
		Categories:        []string{"*"},
		TrustProxyHeaders: true,
	}
}
