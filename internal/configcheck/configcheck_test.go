package configcheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWritableDir(t *testing.T) {
	t.Parallel()

	t.Run("existing writable dir", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, WritableDir(t.TempDir()))
	})

	t.Run("creates nested dir", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(t.TempDir(), "a", "b", "c")
		require.NoError(t, WritableDir(dir))
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("empty path", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, WritableDir(""))
	})

	t.Run("path is a file", func(t *testing.T) {
		t.Parallel()
		f := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(f, []byte("x"), 0o644))
		assert.Error(t, WritableDir(f))
	})

	t.Run("readonly dir", func(t *testing.T) {
		t.Parallel()

		if os.Getuid() == 0 {
			t.Skip("running as root; permission checks are bypassed")
		}

		dir := filepath.Join(t.TempDir(), "ro")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.Chmod(dir, 0o555))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
		assert.Error(t, WritableDir(dir))
	})
}

func TestLogPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	t.Run("file output off skips check", func(t *testing.T) {
		t.Parallel()

		cfg := servercfg.LogConfig{FileOutputLevel: "off", FileRotator: servercfg.FileRotatorConfig{Path: ""}}
		require.NoError(t, LogPath(cfg))
	})

	t.Run("empty level skips check", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, LogPath(servercfg.LogConfig{}))
	})

	t.Run("enabled but no path", func(t *testing.T) {
		t.Parallel()

		cfg := servercfg.LogConfig{FileOutputLevel: "info"}
		assert.ErrorContains(t, LogPath(cfg), "no file rotator path")
	})

	t.Run("enabled and writable", func(t *testing.T) {
		t.Parallel()

		cfg := servercfg.LogConfig{
			FileOutputLevel: "info",
			FileRotator:     servercfg.FileRotatorConfig{Path: filepath.Join(dir, "logs")},
		}
		require.NoError(t, LogPath(cfg))
	})
}

func TestTorrentDir(t *testing.T) {
	t.Parallel()

	require.NoError(t, TorrentDir(filepath.Join(t.TempDir(), "torrents")))
	assert.Error(t, TorrentDir(""))
}

func TestPostgres(t *testing.T) {
	t.Parallel()

	t.Run("unreachable host fails fast", func(t *testing.T) {
		t.Parallel()
		port := freePort(t)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		cfg := postgres.NewDefaultConfig()
		cfg.Host = "127.0.0.1"
		cfg.Port = uint(port)

		err := Postgres(ctx, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot connect to postgres")
	})

	t.Run("invalid config", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		err := Postgres(ctx, postgres.Config{Host: "h", Username: "u", Port: 0, Database: "d", MaxConnections: 0})
		require.Error(t, err)
	})
}

func TestElasticsearch(t *testing.T) {
	t.Parallel()

	t.Run("reachable", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Elastic-Product", "Elasticsearch")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		cfg := elasticsearch.Config{Addresses: []string{srv.URL}}
		require.NoError(t, Elasticsearch(context.Background(), cfg))
	})

	t.Run("unreachable", func(t *testing.T) {
		t.Parallel()
		port := freePort(t)
		cfg := elasticsearch.Config{Addresses: []string{"http://127.0.0.1:" + itoa(port)}}
		err := Elasticsearch(context.Background(), cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot connect to elasticsearch")
	})
}

func TestKafkaBrokers(t *testing.T) {
	t.Parallel()

	t.Run("no brokers", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, KafkaBrokers(context.Background(), nil))
	})

	t.Run("unreachable broker", func(t *testing.T) {
		t.Parallel()
		port := freePort(t)
		err := KafkaBrokers(context.Background(), []string{"127.0.0.1:" + itoa(port)})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "kafka broker")
	})

	t.Run("reachable broker", func(t *testing.T) {
		t.Parallel()

		lc := net.ListenConfig{}

		ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)

		defer ln.Close()

		require.NoError(t, KafkaBrokers(context.Background(), []string{ln.Addr().String()}))
	})

	t.Run("collects all failures", func(t *testing.T) {
		t.Parallel()
		p1 := freePort(t)
		p2 := freePort(t)
		err := KafkaBrokers(context.Background(), []string{"127.0.0.1:" + itoa(p1), "127.0.0.1:" + itoa(p2)})
		require.Error(t, err)
		assert.Equal(t, 2, strings.Count(err.Error(), "is unreachable"))
	})
}

func TestTMDB(t *testing.T) {
	t.Parallel()

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, TMDB(context.Background(), tmdb.Config{Enabled: false, AccessToken: "tok"}))
	})

	t.Run("enabled but no token", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, TMDB(context.Background(), tmdb.Config{Enabled: true, AccessToken: ""}))
	})

	t.Run("enabled with token cannot reach api", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		require.Error(t, TMDB(ctx, tmdb.Config{Enabled: true, AccessToken: "tok", RateLimit: 20}))
	})
}

func TestHTTPEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, HTTPEndpoint(context.Background(), ""))
	})

	t.Run("invalid url", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, HTTPEndpoint(context.Background(), "://bad"))
	})

	t.Run("reachable any status", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		require.NoError(t, HTTPEndpoint(context.Background(), srv.URL))
	})

	t.Run("unreachable", func(t *testing.T) {
		t.Parallel()
		port := freePort(t)
		err := HTTPEndpoint(context.Background(), "http://127.0.0.1:"+itoa(port))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is unreachable")
	})
}

func freePort(t *testing.T) int {
	t.Helper()

	lc := net.ListenConfig{}

	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer ln.Close()

	return ln.Addr().(*net.TCPAddr).Port
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
