package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	require.NoError(t, cfg.Validate())

	t.Run("bad url", func(t *testing.T) {
		t.Parallel()

		cfg := NewDefaultConfig()
		cfg.Urls = []string{"not-a-url", "ftp://x"}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid webhook url")
	})

	t.Run("bad event", func(t *testing.T) {
		t.Parallel()

		cfg := NewDefaultConfig()
		cfg.Events = []string{"new_torrent"}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported webhook event")
	})

	t.Run("bad ranges", func(t *testing.T) {
		t.Parallel()

		cfg := NewDefaultConfig()
		cfg.Timeout = 0
		cfg.MaxRetries = 99
		cfg.QueueSize = 0

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
		assert.Contains(t, err.Error(), "max_retries")
		assert.Contains(t, err.Error(), "queue_size")
	})

	t.Run("bad base url", func(t *testing.T) {
		t.Parallel()

		cfg := NewDefaultConfig()
		cfg.BaseURL = "localhost:3333"

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid base_url")
	})

	t.Run("bad category", func(t *testing.T) {
		t.Parallel()

		cfg := NewDefaultConfig()
		cfg.Categories = []string{"music", "3d"}
		cfg.TitlePatterns = []string{"(?i)[z-a]"}
		cfg.FilenamePatterns = []string{"("}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported webhook category \"3d\"")
		assert.Contains(t, err.Error(), "invalid title pattern")
		assert.Contains(t, err.Error(), "invalid filename pattern")
	})

	t.Run("valid filters", func(t *testing.T) {
		t.Parallel()

		cfg := NewDefaultConfig()
		cfg.Categories = []string{"movie", "tv_show", "music"}
		cfg.TitlePatterns = []string{`\bflac\b`, "2160p"}
		cfg.FilenamePatterns = []string{`\.mkv$`}

		require.NoError(t, cfg.Validate())
	})
}

func TestPublisherDeliversEvent(t *testing.T) {
	t.Parallel()

	var received atomic.Pointer[map[string]any]

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			received.Store(&body)
		}

		w.WriteHeader(http.StatusOK)
	}))

	defer srv.Close()

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{srv.URL}
	cfg.BaseURL = "http://localhost:3333"

	pub := NewPublisher(cfg, testutil.NewTestLogger())

	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{
		Event:       EventClassified,
		InfoHash:    "abcdef1234567890abcdef1234567890abcdef12",
		Name:        "Artist - Album (2022) FLAC",
		Size:        1234,
		ContentType: strPtr("music"),
		Magnet:      "magnet:?xt=urn:btih:abcdef1234567890abcdef1234567890abcdef12",
	})

	require.Eventually(t, func() bool {
		body := received.Load()
		return body != nil
	}, 5*time.Second, 10*time.Millisecond)

	body := *received.Load()
	assert.Equal(t, "classified", body["event"])
	assert.Equal(t, "Artist - Album (2022) FLAC", body["name"])
	assert.Equal(t, "music", body["content_type"])
	assert.Equal(t, "http://localhost:3333/api/torrents/abcdef1234567890abcdef1234567890abcdef12/download", body["torrent_url"])
}

func TestPublisherRespectsEventFilter(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{srv.URL}
	cfg.Events = []string{EventClassified}

	pub := NewPublisher(cfg, testutil.NewTestLogger())

	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: "other"})
	pub.Publish(context.Background(), Event{Event: EventClassified})

	require.Eventually(t, func() bool {
		return hits.Load() == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func TestPublisherDisabled(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := NewDefaultConfig()
	cfg.Enabled = false
	cfg.Urls = []string{srv.URL}

	pub := NewPublisher(cfg, testutil.NewTestLogger())

	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: EventClassified})
	time.Sleep(200 * time.Millisecond)

	assert.Zero(t, hits.Load())
}

func TestPublisherRetriesThenDrops(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{srv.URL}
	cfg.MaxRetries = 2

	pub := NewPublisher(cfg, testutil.NewTestLogger())

	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: EventClassified})

	require.Eventually(t, func() bool {
		return attempts.Load() >= 3
	}, 10*time.Second, 20*time.Millisecond)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestPublisherRestartAfterStop(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{srv.URL}

	pub := NewPublisher(cfg, testutil.NewTestLogger())

	require.NoError(t, pub.Start(context.Background()))
	require.NoError(t, pub.Stop(context.Background()))

	// Events published while stopped are queued and delivered on restart.
	pub.Publish(context.Background(), Event{Event: EventClassified})

	// A fresh Start/Stop cycle must deliver again.
	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: EventClassified})

	require.Eventually(t, func() bool {
		return hits.Load() == 2
	}, 5*time.Second, 10*time.Millisecond)
}

func TestPublisherUpdateQueueSize(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{srv.URL}
	cfg.QueueSize = 4

	pub := NewPublisher(cfg, testutil.NewTestLogger())

	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	// Enqueue, then shrink the queue: pending events must survive the swap.
	pub.Publish(context.Background(), Event{Event: EventClassified, Name: "before"})

	require.Eventually(t, func() bool {
		return hits.Load() == 1
	}, 5*time.Second, 10*time.Millisecond)

	cfg.QueueSize = 1
	pub.Update(cfg)

	pub.Publish(context.Background(), Event{Event: EventClassified, Name: "after"})

	require.Eventually(t, func() bool {
		return hits.Load() == 2
	}, 5*time.Second, 10*time.Millisecond)
}

func TestPublisherSendsHeaders(t *testing.T) {
	t.Parallel()

	var gotHeader atomic.Pointer[string]

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Header.Get("Authorization")
		gotHeader.Store(&v)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{srv.URL}
	cfg.Headers = map[string]string{"Authorization": "Bearer token123"}

	pub := NewPublisher(cfg, testutil.NewTestLogger())

	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: EventClassified})

	require.Eventually(t, func() bool {
		return gotHeader.Load() != nil
	}, 5*time.Second, 10*time.Millisecond)

	assert.Equal(t, "Bearer token123", *gotHeader.Load())
}

func TestPublisherSendsUserAgent(t *testing.T) {
	t.Parallel()

	var gotUA atomic.Pointer[string]

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Header.Get("User-Agent")
		gotUA.Store(&v)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{srv.URL}

	pub := NewPublisher(cfg, testutil.NewTestLogger())

	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: EventClassified})

	require.Eventually(t, func() bool {
		return gotUA.Load() != nil
	}, 5*time.Second, 10*time.Millisecond)

	assert.Equal(t, version.UserAgent(), *gotUA.Load())
}

func TestPublisherSnapshotDispatch(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{srv.URL}

	pub := NewPublisher(cfg, testutil.NewTestLogger())

	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	// Publish, then immediately disable: the already-accepted event is still
	// delivered because the delivery settings were captured at publish time.
	pub.Publish(context.Background(), Event{Event: EventClassified})

	cfg.Enabled = false
	pub.Update(cfg)

	require.Eventually(t, func() bool {
		return hits.Load() == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func TestPublisherUpdateConfig(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := NewDefaultConfig()
	cfg.Enabled = false

	pub := NewPublisher(cfg, testutil.NewTestLogger())

	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: EventClassified})
	time.Sleep(100 * time.Millisecond)
	assert.Zero(t, hits.Load())

	cfg2 := NewDefaultConfig()
	cfg2.Enabled = true
	cfg2.Urls = []string{srv.URL}
	pub.Update(cfg2)

	pub.Publish(context.Background(), Event{Event: EventClassified})

	require.Eventually(t, func() bool {
		return hits.Load() == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func TestNewClassifiedEvent(t *testing.T) {
	t.Parallel()

	e := NewClassifiedEvent(sampleTorrent())

	assert.Equal(t, EventClassified, e.Event)
	assert.Equal(t, "abcdef1234567890abcdef1234567890abcdef12", e.InfoHash)
	assert.Equal(t, "Artist - Album (2022) FLAC", e.Name)
	assert.Equal(t, int64(999), e.Size)
	assert.Equal(t, "music", *e.ContentType)
	assert.Equal(t, "en", e.Languages[0])
	assert.Equal(t, int32(5), *e.Seeders)
	assert.Contains(t, e.Magnet, "magnet:?")
	assert.Equal(t, []string{
		"Artist - Album (2022) FLAC/01 - Track.flac",
		"Artist - Album (2022) FLAC/cover.jpg",
	}, e.Files)
}

func TestPublisherFiltersByCategory(t *testing.T) {
	t.Parallel()

	hits := newCountingServer(t)

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{hits.url}
	cfg.Categories = []string{"music"}

	pub := NewPublisher(cfg, testutil.NewTestLogger())
	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: EventClassified, ContentType: strPtr("music")})
	pub.Publish(context.Background(), Event{Event: EventClassified, ContentType: strPtr("movie")})
	pub.Publish(context.Background(), Event{Event: EventClassified})

	require.Eventually(t, func() bool {
		return hits.load() == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func TestPublisherFiltersByTitlePattern(t *testing.T) {
	t.Parallel()

	hits := newCountingServer(t)

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{hits.url}
	cfg.TitlePatterns = []string{`\bflac\b`}

	pub := NewPublisher(cfg, testutil.NewTestLogger())
	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: EventClassified, Name: "Artist - Album (2022) FLAC"})
	pub.Publish(context.Background(), Event{Event: EventClassified, Name: "Artist - Album (2022) MP3"})
	pub.Publish(context.Background(), Event{Event: EventClassified, Name: "Artist - Album (2022) flac"})

	require.Eventually(t, func() bool {
		return hits.load() == 2
	}, 5*time.Second, 10*time.Millisecond)
}

func TestPublisherFiltersByFilenamePattern(t *testing.T) {
	t.Parallel()

	hits := newCountingServer(t)

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{hits.url}
	cfg.FilenamePatterns = []string{`\.flac$`}

	pub := NewPublisher(cfg, testutil.NewTestLogger())
	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: EventClassified, Files: []string{"Album/01 - Track.flac"}})
	pub.Publish(context.Background(), Event{Event: EventClassified, Files: []string{"Album/01 - Track.mp3"}})
	pub.Publish(context.Background(), Event{Event: EventClassified})

	require.Eventually(t, func() bool {
		return hits.load() == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func TestPublisherFilterCombination(t *testing.T) {
	t.Parallel()

	hits := newCountingServer(t)

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{hits.url}
	cfg.Categories = []string{"music"}
	cfg.TitlePatterns = []string{`\bflac\b`}
	cfg.FilenamePatterns = []string{`\.flac$`}

	pub := NewPublisher(cfg, testutil.NewTestLogger())
	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{
		Event:       EventClassified,
		Name:        "Artist - Album (2022) FLAC",
		ContentType: strPtr("music"),
		Files:       []string{"Album/01 - Track.flac"},
	})
	pub.Publish(context.Background(), Event{
		Event:       EventClassified,
		Name:        "Artist - Album (2022) FLAC",
		ContentType: strPtr("movie"),
		Files:       []string{"Album/01 - Track.flac"},
	})
	pub.Publish(context.Background(), Event{
		Event:       EventClassified,
		Name:        "Artist - Album (2022) MP3",
		ContentType: strPtr("music"),
		Files:       []string{"Album/01 - Track.flac"},
	})

	require.Eventually(t, func() bool {
		return hits.load() == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func TestPublisherUpdateRecompilesFilters(t *testing.T) {
	t.Parallel()

	hits := newCountingServer(t)

	cfg := NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{hits.url}
	cfg.Categories = []string{"music"}

	pub := NewPublisher(cfg, testutil.NewTestLogger())
	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	pub.Publish(context.Background(), Event{Event: EventClassified, ContentType: strPtr("movie")})
	time.Sleep(100 * time.Millisecond)
	assert.Zero(t, hits.load())

	cfg2 := NewDefaultConfig()
	cfg2.Enabled = true
	cfg2.Urls = []string{hits.url}
	cfg2.Categories = []string{"movie"}
	pub.Update(cfg2)

	pub.Publish(context.Background(), Event{Event: EventClassified, ContentType: strPtr("movie")})

	require.Eventually(t, func() bool {
		return hits.load() == 1
	}, 5*time.Second, 10*time.Millisecond)
}

type countingServer struct {
	url  string
	hits *atomic.Int32
}

func newCountingServer(t *testing.T) *countingServer {
	t.Helper()

	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))

	t.Cleanup(srv.Close)

	return &countingServer{url: srv.URL, hits: &hits}
}

func (s *countingServer) load() int32 {
	return s.hits.Load()
}

func strPtr(s string) *string { return &s }
