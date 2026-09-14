package processor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/blocking"
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/queue"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/hexsans/hexmagnet/internal/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCompileFilterState_Empty(t *testing.T) {
	t.Parallel()

	cfg := classifier.TorrentFilterConfig{
		Mode:             classifier.TorrentFilterOff,
		TitlePatterns:    []string{},
		FilenamePatterns: []string{},
	}

	fs, err := compileFilterState(cfg)
	require.NoError(t, err)
	assert.Equal(t, classifier.TorrentFilterOff, fs.mode)
	assert.Empty(t, fs.titlePatterns)
	assert.Empty(t, fs.filenamePatterns)
}

func TestCompileFilterState_ValidPatterns(t *testing.T) {
	t.Parallel()

	cfg := classifier.TorrentFilterConfig{
		Mode:             classifier.TorrentFilterDiscard,
		TitlePatterns:    []string{"xxx", "porn"},
		FilenamePatterns: []string{".*\\.exe$", "crack"},
	}

	fs, err := compileFilterState(cfg)
	require.NoError(t, err)
	require.Len(t, fs.titlePatterns, 2)
	require.Len(t, fs.filenamePatterns, 2)

	assert.True(t, fs.titlePatterns[0].MatchString("XXX video"))
	assert.True(t, fs.filenamePatterns[0].MatchString("setup.exe"))
}

func TestCompileFilterState_InvalidPattern(t *testing.T) {
	t.Parallel()

	cfg := classifier.TorrentFilterConfig{
		Mode:          classifier.TorrentFilterDiscard,
		TitlePatterns: []string{"["},
	}

	_, err := compileFilterState(cfg)
	assert.ErrorContains(t, err, "invalid title pattern")
}

func TestNewTorrentContent_WithoutContent(t *testing.T) {
	t.Parallel()

	torrent := model.Torrent{
		InfoHash:   testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		Name:       "Test Torrent",
		Size:       1000,
		CreatedAt:  time.Now(),
		FilesCount: model.NullUint{Valid: true, Uint: 5},
	}

	cl := classifier.ClassificationResult{
		ContentAttributes: classifier.ContentAttributes{
			ContentType: model.NewNullContentType(model.ContentTypeMovie),
			Languages:   model.Languages{"en": {}},
		},
	}

	result := newTorrentContent(torrent, cl, 30)
	assert.Equal(t, torrent.InfoHash, result.InfoHash)
	assert.Equal(t, torrent.Name, result.Name)
	assert.True(t, result.ContentType.Valid)
	assert.Equal(t, model.ContentTypeMovie, result.ContentType.ContentType)
	assert.Contains(t, result.Languages, model.Language("en"))
}

func TestNewTorrentContent_WithContent(t *testing.T) {
	t.Parallel()

	torrent := model.Torrent{
		InfoHash: testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		Name:     "Test Torrent",
		Size:     1000,
	}

	content := &model.Content{
		Type:   model.ContentTypeMovie,
		Source: "tmdb",
		ID:     "123",
		Title:  "Test Movie",
	}

	cl := classifier.ClassificationResult{
		ContentAttributes: classifier.ContentAttributes{
			ContentType: model.NewNullContentType(model.ContentTypeMovie),
		},
	}
	cl.AttachContent(content)

	result := newTorrentContent(torrent, cl, 30)
	assert.True(t, result.ContentSource.Valid)
	assert.Equal(t, "tmdb", result.ContentSource.String)
	assert.True(t, result.ContentID.Valid)
	assert.Equal(t, "123", result.ContentID.String)
	assert.Equal(t, "Test Movie", result.Content.Title)
}

func newTestProcessor() *processor {
	return &processor{logger: zap.NewNop().Sugar()}
}

func TestFilteredByTorrentFilter_Off(t *testing.T) {
	t.Parallel()

	p := newTestProcessor()
	p.filter.Store(&filterState{mode: classifier.TorrentFilterOff})

	result := p.filteredByTorrentFilter(model.Torrent{Name: "anything"})
	assert.False(t, result)
}

func TestFilteredByTorrentFilter_DiscardMatch(t *testing.T) {
	t.Parallel()

	p := newTestProcessor()
	fs, err := compileFilterState(classifier.TorrentFilterConfig{
		Mode:          classifier.TorrentFilterDiscard,
		TitlePatterns: []string{"xxx"},
	})
	require.NoError(t, err)
	p.filter.Store(fs)

	result := p.filteredByTorrentFilter(model.Torrent{Name: "XXX video"})
	assert.True(t, result)
}

func TestFilteredByTorrentFilter_DiscardNoMatch(t *testing.T) {
	t.Parallel()

	p := newTestProcessor()
	fs, err := compileFilterState(classifier.TorrentFilterConfig{
		Mode:          classifier.TorrentFilterDiscard,
		TitlePatterns: []string{"xxx"},
	})
	require.NoError(t, err)
	p.filter.Store(fs)

	result := p.filteredByTorrentFilter(model.Torrent{Name: "clean movie"})
	assert.False(t, result)
}

func TestFilteredByTorrentFilter_ProcessMatch(t *testing.T) {
	t.Parallel()

	p := newTestProcessor()
	fs, err := compileFilterState(classifier.TorrentFilterConfig{
		Mode:          classifier.TorrentFilterProcess,
		TitlePatterns: []string{"linux"},
	})
	require.NoError(t, err)
	p.filter.Store(fs)

	result := p.filteredByTorrentFilter(model.Torrent{Name: "Ubuntu Linux ISO"})
	assert.False(t, result)
}

func TestFilteredByTorrentFilter_ProcessNoMatch(t *testing.T) {
	t.Parallel()

	p := newTestProcessor()
	fs, err := compileFilterState(classifier.TorrentFilterConfig{
		Mode:          classifier.TorrentFilterProcess,
		TitlePatterns: []string{"linux"},
	})
	require.NoError(t, err)
	p.filter.Store(fs)

	result := p.filteredByTorrentFilter(model.Torrent{Name: "Windows ISO"})
	assert.True(t, result)
}

func TestFilteredByTorrentFilter_FilenameMatch(t *testing.T) {
	t.Parallel()

	p := newTestProcessor()
	fs, err := compileFilterState(classifier.TorrentFilterConfig{
		Mode:             classifier.TorrentFilterDiscard,
		FilenamePatterns: []string{".*\\.exe$"},
	})
	require.NoError(t, err)
	p.filter.Store(fs)

	torrent := model.Torrent{
		Name: "Software Bundle",
		Files: []model.TorrentFile{
			{PathParts: []string{"setup.exe"}, Extension: model.NewNullString("exe"), Size: 100},
		},
	}

	result := p.filteredByTorrentFilter(torrent)
	assert.True(t, result)
}

func TestFilteredByTorrentFilter_TitleMatchPrecedesFilename(t *testing.T) {
	t.Parallel()

	p := newTestProcessor()
	fs, err := compileFilterState(classifier.TorrentFilterConfig{
		Mode:             classifier.TorrentFilterDiscard,
		TitlePatterns:    []string{"clean"},
		FilenamePatterns: []string{".*\\.exe$"},
	})
	require.NoError(t, err)
	p.filter.Store(fs)

	torrent := model.Torrent{
		Name: "Clean title",
		Files: []model.TorrentFile{
			{PathParts: []string{"file.txt"}, Extension: model.NewNullString("txt"), Size: 100},
		},
	}

	result := p.filteredByTorrentFilter(torrent)
	assert.True(t, result)
}

func TestRecentlyClassified(t *testing.T) {
	t.Parallel()

	p := &processor{}
	ih := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	assert.False(t, p.wasRecentlyClassified(ih))

	p.markRecentlyClassified(ih)
	assert.True(t, p.wasRecentlyClassified(ih))
}

func TestRecentlyClassified_DifferentHash(t *testing.T) {
	t.Parallel()

	p := &processor{}
	ih1 := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	ih2 := testutil.MustParseID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	p.markRecentlyClassified(ih1)
	assert.True(t, p.wasRecentlyClassified(ih1))
	assert.False(t, p.wasRecentlyClassified(ih2))
}

func TestRecentlyClassified_Expired(t *testing.T) {
	t.Parallel()

	p := &processor{}
	ih := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	p.recentlyClassified.Store(ih, time.Now().Add(-5*time.Minute))

	assert.False(t, p.wasRecentlyClassified(ih))
}

func TestRecentlyClassified_WrongType(t *testing.T) {
	t.Parallel()

	p := &processor{}
	ih := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	p.recentlyClassified.Store(ih, "not a time")

	assert.False(t, p.wasRecentlyClassified(ih))
}

func TestUpdateTorrentFilter(t *testing.T) {
	t.Parallel()

	p := &processor{
		filter: atomic.Pointer[filterState]{},
		logger: zap.NewNop().Sugar(),
	}

	err := p.UpdateTorrentFilter(classifier.TorrentFilterConfig{
		Mode:             classifier.TorrentFilterDiscard,
		TitlePatterns:    []string{"bad"},
		FilenamePatterns: []string{},
	})
	require.NoError(t, err)

	fs := p.filter.Load()
	require.NotNil(t, fs)
	assert.Equal(t, classifier.TorrentFilterDiscard, fs.mode)
	assert.Len(t, fs.titlePatterns, 1)
}

func TestUpdateTorrentFilter_InvalidPattern(t *testing.T) {
	t.Parallel()

	p := &processor{
		filter: atomic.Pointer[filterState]{},
		logger: zap.NewNop().Sugar(),
	}

	err := p.UpdateTorrentFilter(classifier.TorrentFilterConfig{
		Mode:          classifier.TorrentFilterDiscard,
		TitlePatterns: []string{"["},
	})
	assert.Error(t, err)
}

type mockProducer struct{}

func (mockProducer) Produce(_ string, _ string, _ any) {}
func (mockProducer) Close() error                      { return nil }

type recordingProducer struct {
	topics []string
}

func (p *recordingProducer) Produce(topic, _ string, _ any) {
	p.topics = append(p.topics, topic)
}

func (*recordingProducer) Close() error { return nil }

func TestHandleClassified_DropsMissingHashes(t *testing.T) {
	t.Parallel()

	prod := &recordingProducer{}
	p := &processor{
		kafkaProducer: prod,
		logger:        zap.NewNop().Sugar(),
	}

	missing := []protocol.ID{testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}

	err := p.handleClassified(context.Background(), nil, missing, nil, MessageParams{})
	require.NoError(t, err)
	assert.Empty(t, prod.topics, "missing torrents must not be re-produced")
}

func TestSplitClassifyFailures_PausesOnLLMFailure(t *testing.T) {
	t.Parallel()

	llmErr := &classifier.LLMClassifyError{Cause: assert.AnError}
	failures := []classifyFailure{
		{infoHash: testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), err: errors.New("db error")},
		{infoHash: testutil.MustParseID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"), err: llmErr},
	}

	retryable, pauseErr := splitClassifyFailures(failures)

	require.Len(t, retryable, 1)
	require.NotErrorIs(t, retryable[0].err, llmErr)
	require.Error(t, pauseErr)
	require.ErrorIs(t, pauseErr, assert.AnError)
}

func TestHandleClassified_PausesOnLLMFailure(t *testing.T) {
	t.Parallel()

	p := &processor{logger: zap.NewNop().Sugar()}

	err := p.handleClassified(context.Background(), nil, nil, []classifyFailure{
		{
			infoHash: testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			err:      &classifier.LLMClassifyError{Cause: assert.AnError},
		},
	}, MessageParams{})

	require.Error(t, err)

	var llmErr *classifier.LLMClassifyError
	require.ErrorAs(t, err, &llmErr)
}

func TestNew_ReturnsResult(t *testing.T) {
	t.Parallel()

	p := Params{
		ClassifierConfig: classifier.Config{},
		SearchRuntime:    dbsearch.NewRuntime(indexer.SearchConfig{Backend: "postgresql"}),
		Workflow:         utils.NewLazy(func() (classifier.Runner, error) { return nil, errors.New("unused in test") }),
		Queries:          utils.NewLazy(func() (*db.Queries, error) { return nil, errors.New("unused in test") }),
		BlockingManager:  utils.NewLazy(func() (blocking.Manager, error) { return nil, errors.New("unused in test") }),
		Producer:         mockProducer{},
		Logger:           zap.NewNop().Sugar(),
	}

	result := New(p)
	assert.NotNil(t, result.Processor)
}

func TestNewConsumer_ReturnsResult(t *testing.T) {
	t.Parallel()

	r := queue.NewRuntime(queue.Config{Backend: "memory"}, zap.NewNop().Sugar())

	p := ConsumerParams{
		ConsumerMaker: r.DynamicConsumerMaker(),
		Logger:        zap.NewNop().Sugar(),
		Processor:     utils.NewLazy(func() (Processor, error) { return nil, errors.New("unused in test") }),
		Runtime:       r,
	}

	result := NewConsumer(p)
	assert.NotNil(t, result.Worker)
	assert.Equal(t, "message_queue_process_consumer", result.Worker.Key())
}

func TestNewProcessorFxModule(t *testing.T) {
	t.Parallel()

	mod := NewProcessorFxModule()
	assert.NotNil(t, mod)
}

func TestPublishClassified_SendsEvent(t *testing.T) {
	t.Parallel()

	var received atomic.Value

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			received.Store(body)
		}

		w.WriteHeader(http.StatusOK)
	}))

	defer srv.Close()

	cfg := webhook.NewDefaultConfig()
	cfg.Enabled = true
	cfg.Urls = []string{srv.URL}

	pub := webhook.NewPublisher(cfg, zap.NewNop().Sugar())
	require.NoError(t, pub.Start(context.Background()))

	defer func() { _ = pub.Stop(context.Background()) }()

	p := newTestProcessor()
	p.webhook = pub

	torrents := []model.Torrent{sampleTorrentForProcessor()}

	p.publishClassified(context.Background(), torrents)

	require.Eventually(t, func() bool {
		return received.Load() != nil
	}, 5*time.Second, 10*time.Millisecond)

	body := received.Load().(map[string]any)
	assert.Equal(t, "classified", body["event"])
	assert.Equal(t, "Artist - Album (2022) FLAC", body["name"])
}

func TestPublishClassified_NilPublisher(t *testing.T) {
	t.Parallel()

	p := newTestProcessor()
	p.publishClassified(context.Background(), []model.Torrent{sampleTorrentForProcessor()})
}

func TestPublishClassified_EmptyTorrents(t *testing.T) {
	t.Parallel()

	p := newTestProcessor()
	p.publishClassified(context.Background(), nil)
}

func sampleTorrentForProcessor() model.Torrent {
	return model.Torrent{
		InfoHash:    testutil.MustParseID("abcdef1234567890abcdef1234567890abcdef12"),
		Name:        "Artist - Album (2022) FLAC",
		Size:        999,
		ContentType: model.NewNullContentType("music"),
	}
}
