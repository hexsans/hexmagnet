package search

import (
	"context"
	"testing"

	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/stretchr/testify/assert"
)

func TestNewRuntime_Defaults(t *testing.T) {
	t.Parallel()

	cfg := indexer.SearchConfig{Backend: "postgresql"}
	r := NewRuntime(cfg)

	assert.Equal(t, "postgresql", r.Backend.Get())
	assert.Equal(t, cfg, r.SearchConfig.Get())
}

func TestNewRuntime_Elasticsearch(t *testing.T) {
	t.Parallel()

	cfg := indexer.SearchConfig{
		Backend: "elasticsearch",
		Elasticsearch: elasticsearch.Config{
			Addresses: []string{"http://es:9200"},
		},
	}
	r := NewRuntime(cfg)

	assert.Equal(t, "elasticsearch", r.Backend.Get())
	assert.Equal(t, cfg, r.SearchConfig.Get())
}

func TestRuntime_SwitchBackend_UpdatesBackend(t *testing.T) {
	t.Parallel()

	r := NewRuntime(indexer.SearchConfig{Backend: "postgresql"})

	newCfg := indexer.SearchConfig{Backend: "elasticsearch"}
	r.SwitchBackend(newCfg)

	assert.Equal(t, "elasticsearch", r.Backend.Get())
}

func TestRuntime_SwitchBackend_UpdatesConfig(t *testing.T) {
	t.Parallel()

	r := NewRuntime(indexer.SearchConfig{Backend: "postgresql"})

	newCfg := indexer.SearchConfig{
		Backend: "elasticsearch",
		Elasticsearch: elasticsearch.Config{
			Addresses: []string{"http://new-es:9200"},
		},
	}
	r.SwitchBackend(newCfg)

	assert.Equal(t, "elasticsearch", r.Backend.Get())
	assert.Equal(t, newCfg, r.SearchConfig.Get())
}

func TestRuntime_SwitchBackend_SameBackend(t *testing.T) {
	t.Parallel()

	r := NewRuntime(indexer.SearchConfig{Backend: "postgresql"})
	r.SwitchBackend(indexer.SearchConfig{Backend: "postgresql"})

	assert.Equal(t, "postgresql", r.Backend.Get())
}

func TestRuntime_Search_InitiallyNil(t *testing.T) {
	t.Parallel()

	r := NewRuntime(indexer.SearchConfig{Backend: "postgresql"})

	assert.Nil(t, r.Search.Get(), "Search impl should be nil until set")
}

func TestRuntime_Search_SetAndGet(t *testing.T) {
	t.Parallel()

	r := NewRuntime(indexer.SearchConfig{Backend: "postgresql"})

	r.Search.Set(new(mockSearch))
	assert.NotNil(t, r.Search.Get())
}

func TestRuntime_SwitchBackend_CallsFactory(t *testing.T) {
	t.Parallel()

	var capturedCfg indexer.SearchConfig

	r := NewRuntime(indexer.SearchConfig{Backend: "postgresql"})
	r.SetSearchFactory(func(cfg indexer.SearchConfig) Search {
		capturedCfg = cfg
		return new(mockSearch)
	})

	newCfg := indexer.SearchConfig{Backend: "elasticsearch"}
	r.SwitchBackend(newCfg)

	assert.Equal(t, "elasticsearch", r.Backend.Get())
	assert.NotNil(t, r.Search.Get(), "Search should be set by factory after SwitchBackend")
	assert.Equal(t, newCfg, capturedCfg, "factory should be called with the new config")
}

func TestRuntime_SwitchBackend_FactoryOverridesPrevious(t *testing.T) {
	t.Parallel()

	var called bool

	r := NewRuntime(indexer.SearchConfig{Backend: "postgresql"})
	r.Search.Set(new(mockSearch))

	r.SetSearchFactory(func(_ indexer.SearchConfig) Search {
		called = true
		return new(mockSearch)
	})

	r.SwitchBackend(indexer.SearchConfig{Backend: "elasticsearch"})

	assert.True(t, called, "factory should be called")
	assert.NotNil(t, r.Search.Get(), "Search should be replaced by factory output")
}

func TestRuntime_SwitchBackend_FactoryWithSameConfig(t *testing.T) {
	t.Parallel()

	var factoryCalls int

	r := NewRuntime(indexer.SearchConfig{Backend: "postgresql"})
	r.SetSearchFactory(func(_ indexer.SearchConfig) Search {
		factoryCalls++
		return new(mockSearch)
	})

	r.SwitchBackend(indexer.SearchConfig{Backend: "postgresql"})
	assert.Equal(t, 1, factoryCalls)
	assert.NotNil(t, r.Search.Get())
}

func TestRuntime_SwitchBackend_NoFactoryDoesNotPanic(t *testing.T) {
	t.Parallel()

	r := NewRuntime(indexer.SearchConfig{Backend: "postgresql"})

	r.SwitchBackend(indexer.SearchConfig{Backend: "elasticsearch"})

	assert.Equal(t, "elasticsearch", r.Backend.Get())
	assert.Nil(t, r.Search.Get(), "Search should remain nil when no factory is set")
}

type mockSearch struct{}

func (*mockSearch) TorrentSearch(_ context.Context, _ TorrentSearchParams) (TorrentSearchResult, error) {
	return TorrentSearchResult{}, nil
}

func (*mockSearch) TorrentsWithMissingInfoHashes(
	_ context.Context,
	_ TorrentsWithMissingInfoHashesParams,
) (TorrentsWithMissingInfoHashesResult, error) {
	return TorrentsWithMissingInfoHashesResult{}, nil
}

func (*mockSearch) TorrentFiles(_ context.Context, _ TorrentFilesSearchParams) (TorrentFilesResult, error) {
	return TorrentFilesResult{}, nil
}
func (*mockSearch) Close() error { return nil }
