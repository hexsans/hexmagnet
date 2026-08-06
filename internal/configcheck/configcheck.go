package configcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"go.uber.org/zap"
)

const (
	postgresTimeout      = 5 * time.Second
	elasticsearchTimeout = 5 * time.Second
	kafkaDialTimeout     = 2 * time.Second
	tmdbTimeout          = 10 * time.Second
	endpointTimeout      = 3 * time.Second
)

// WritableDir ensures dir exists and is writable, creating it if needed.
func WritableDir(dir string) error {
	if dir == "" {
		return errors.New("path is empty")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create directory %q: %w", dir, err)
	}

	probe := filepath.Join(dir, ".hexmagnet-write-probe")
	if err := os.WriteFile(probe, []byte("probe"), 0o644); err != nil {
		return fmt.Errorf("directory %q is not writable: %w", dir, err)
	}

	_ = os.Remove(probe)

	return nil
}

// LogPath validates the file output path when file logging is enabled.
func LogPath(cfg servercfg.LogConfig) error {
	if cfg.FileOutputLevel == "" || cfg.FileOutputLevel == "off" {
		return nil
	}

	if cfg.FileRotator.Path == "" {
		return errors.New("file output is enabled but no file rotator path is set")
	}

	return WritableDir(cfg.FileRotator.Path)
}

// TorrentDir validates the torrent file storage path.
func TorrentDir(path string) error {
	return WritableDir(path)
}

// Postgres checks that a connection to the database can be established.
// No state is mutated.
func Postgres(ctx context.Context, cfg postgres.Config) error {
	ctx, cancel := context.WithTimeout(ctx, postgresTimeout)
	defer cancel()

	if err := postgres.ValidateConnection(ctx, cfg, zap.NewNop().Sugar()); err != nil {
		return fmt.Errorf("cannot connect to postgres at %s:%d/%s: %w", cfg.Host, cfg.Port, cfg.Database, err)
	}

	return nil
}

// Elasticsearch checks that a connection to the cluster can be established.
// No state is mutated.
func Elasticsearch(ctx context.Context, cfg elasticsearch.Config) error {
	ctx, cancel := context.WithTimeout(ctx, elasticsearchTimeout)
	defer cancel()

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("cannot create elasticsearch client: %w", err)
	}

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("cannot connect to elasticsearch at %s: %w", strings.Join(cfg.Addresses, ", "), err)
	}

	return nil
}

// KafkaBrokers checks that every unique broker address accepts TCP
// connections.
func KafkaBrokers(ctx context.Context, brokers []string) error {
	if len(brokers) == 0 {
		return errors.New("no kafka brokers configured")
	}

	ctx, cancel := context.WithTimeout(ctx, kafkaDialTimeout)
	defer cancel()

	dialer := &net.Dialer{Timeout: kafkaDialTimeout}
	seen := map[string]bool{}

	var errs []error

	for _, b := range brokers {
		if seen[b] {
			continue
		}

		seen[b] = true

		conn, err := dialer.DialContext(ctx, "tcp", b)
		if err != nil {
			errs = append(errs, fmt.Errorf("kafka broker %q is unreachable: %w", b, err))
			continue
		}

		_ = conn.Close()
	}

	if len(errs) > 0 {
		return fmt.Errorf("cannot reach kafka brokers: %w", errors.Join(errs...))
	}

	return nil
}

// TMDB validates the access token against the TMDB API. No state is mutated.
func TMDB(ctx context.Context, cfg tmdb.Config) error {
	if !cfg.Enabled || cfg.AccessToken == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, tmdbTimeout)
	defer cancel()

	if err := tmdb.ValidateAccessToken(ctx, cfg, zap.NewNop().Sugar()); err != nil {
		return fmt.Errorf("tmdb: %w", err)
	}

	return nil
}

// HTTPEndpoint checks that an endpoint is reachable. Any HTTP response is
// considered reachable; only transport-level failures are reported.
func HTTPEndpoint(ctx context.Context, endpoint string) error {
	if strings.TrimSpace(endpoint) == "" {
		return errors.New("endpoint URL is empty")
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL %q: %w", endpoint, err)
	}

	if u.Host == "" {
		return fmt.Errorf("endpoint URL %q has no host", endpoint)
	}

	ctx, cancel := context.WithTimeout(ctx, endpointTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL %q: %w", endpoint, err)
	}

	client := &http.Client{Timeout: endpointTimeout}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("endpoint %q is unreachable: %w", endpoint, err)
	}

	_ = res.Body.Close()

	return nil
}
