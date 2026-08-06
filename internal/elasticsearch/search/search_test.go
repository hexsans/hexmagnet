package esearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEmbedder struct {
	embedFunc func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return m.embedFunc(ctx, texts)
}

func (m *mockEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vectors, err := m.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(vectors) == 0 {
		return nil, nil
	}

	return vectors[0], nil
}

var _ embedding.Embedder = (*mockEmbedder)(nil)

func TestNew(t *testing.T) {
	t.Parallel()

	s := New(nil, nil, nil, nil, false)
	assert.NotNil(t, s)
}

func TestTorrentSearch_QueryString(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/torrent_content/_search", r.URL.Path)

		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Contains(t, reqBody, "query")

		testutil.ESOK(w, `{
			"took": 3,
			"timed_out": false,
			"hits": {
				"total": {"value":1,"relation":"eq"},
				"hits": [
					{"_index":"torrent_content","_id":"1","_score":1.0,"_source":{
						"info_hash":"abc123",
						"name":"test torrent",
						"size":1024,
						"seeders":10,
						"leechers":5,
						"files_count":3,
												"created_at":"2024-01-15T10:00:00Z",
						"updated_at":"2024-01-15T10:00:00Z",
						"languages":["en"],
						"sources":["public"],
						"private":false,
						"content_type":"movie",
						"content_source":"tmdb",
						"content_id":"123",
						"title":"A Movie",
						"overview":"Great movie"
					}}
				]
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	result, err := s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		QueryString: "test movie",
	})
	require.NoError(t, err)

	assert.Len(t, result.Items, 1)
	assert.Equal(t, uint(1), result.TotalCount)
	assert.False(t, result.TotalCountIsEstimate)
	assert.False(t, result.HasNextPage)

	item := result.Items[0]
	assert.Equal(t, "abc123", item.InfoHash)
	assert.Equal(t, "test torrent", item.TorrentName)
	assert.Equal(t, int64(1024), item.Size)
	assert.NotNil(t, item.Seeders)
	assert.Equal(t, int32(10), *item.Seeders)
	assert.NotNil(t, item.Leechers)
	assert.Equal(t, int32(5), *item.Leechers)
	assert.NotNil(t, item.FilesCount)
	assert.Equal(t, int32(3), *item.FilesCount)
	assert.NotNil(t, item.ContentType)
	assert.Equal(t, "movie", *item.ContentType)
	assert.NotNil(t, item.ContentSource)
	assert.Equal(t, "tmdb", *item.ContentSource)
	assert.NotNil(t, item.ContentID)
	assert.Equal(t, "123", *item.ContentID)
	assert.NotNil(t, item.ContentTitle)
	assert.Equal(t, "A Movie", *item.ContentTitle)
	assert.NotNil(t, item.ContentOverview)
	assert.Equal(t, "Great movie", *item.ContentOverview)
	assert.False(t, item.TorrentPrivate)
}

func TestTorrentSearch_NoQuery(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		query, ok := reqBody["query"].(map[string]any)
		assert.True(t, ok)
		boolQ, ok := query["bool"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, boolQ, "filter")
		assert.NotContains(t, boolQ, "must")

		testutil.ESOK(w, `{
			"took": 1,
			"timed_out": false,
			"hits": {
				"total": {"value":0,"relation":"eq"},
				"hits": []
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	result, err := s.TorrentSearch(context.Background(), search.TorrentSearchParams{})
	require.NoError(t, err)
	assert.Empty(t, result.Items)
	assert.Equal(t, uint(0), result.TotalCount)
}

func TestTorrentSearch_WithFilters(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		query, ok := reqBody["query"].(map[string]any)
		assert.True(t, ok)
		boolQ, ok := query["bool"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, boolQ, "filter")

		testutil.ESOK(w, `{
			"took": 1,
			"timed_out": false,
			"hits": {
				"total": {"value":0,"relation":"eq"},
				"hits": []
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	_, err = s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		InfoHashes:     []string{"abc123"},
		ContentTypes:   []string{"movie"},
		TorrentSources: []string{"public"},
		Languages:      []string{"en"},
		ReleaseYears:   []int32{2024},
	})
	require.NoError(t, err)
}

func TestTorrentSearch_WithContentRefs(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		query, ok := reqBody["query"].(map[string]any)
		assert.True(t, ok)
		boolQ, ok := query["bool"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, boolQ, "filter")

		testutil.ESOK(w, `{
			"took": 1,
			"timed_out": false,
			"hits": {
				"total": {"value":0,"relation":"eq"},
				"hits": []
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	_, err = s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		ContentRefs: []search.ContentRef{
			{Type: "movie", Source: "tmdb", ID: "123"},
		},
	})
	require.NoError(t, err)
}

func TestTorrentSearch_WithFacets(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Contains(t, reqBody, "aggs")

		testutil.ESOK(w, `{
			"took": 2,
			"timed_out": false,
			"hits": {
				"total": {"value":0,"relation":"eq"},
				"hits": []
			},
			"aggregations": {
				"content_type": {
					"doc_count_error_upper_bound": 0,
					"sum_other_doc_count": 0,
					"buckets": [
						{"key": "movie", "doc_count": 5},
						{"key": "tv", "doc_count": 3}
					]
				},
				"language": {
					"buckets": [
						{"key": "en", "doc_count": 8}
					]
				}
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	result, err := s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		FacetAggregate: search.FacetAggregationConfig{
			ContentType: true,
			Language:    true,
		},
	})
	require.NoError(t, err)
	require.Contains(t, result.Aggregations, "content_type")
	require.Contains(t, result.Aggregations["content_type"].Items, "movie")
	assert.Equal(t, uint(5), result.Aggregations["content_type"].Items["movie"].Count)
	require.Contains(t, result.Aggregations, "language")
}

func TestTorrentSearch_WithOrderBy(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		sort, ok := reqBody["sort"].([]any)
		assert.True(t, ok)
		assert.Len(t, sort, 3)
		entry0, ok := sort[0].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, entry0, "size")

		entry1, ok := sort[1].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, entry1, "created_at")

		entry2, ok := sort[2].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, entry2, "info_hash")

		testutil.ESOK(w, `{
			"took": 1,
			"timed_out": false,
			"hits": {
				"total": {"value":0,"relation":"eq"},
				"hits": []
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	_, err = s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		OrderBy: []search.TorrentSearchOrder{
			{Field: search.FieldSize, Direction: search.SortDesc},
		},
	})
	require.NoError(t, err)
}

func TestTorrentSearch_Pagination(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		assert.Equal(t, true, reqBody["track_total_hits"])

		assert.InEpsilon(t, 20, reqBody["from"], 0.0001)
		assert.InEpsilon(t, 10, reqBody["size"], 0.0001)

		testutil.ESOK(w, `{
			"took": 1,
			"timed_out": false,
			"hits": {
				"total": {"value":0,"relation":"eq"},
				"hits": []
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	_, err = s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		Offset: 20,
		Limit:  10,
	})
	require.NoError(t, err)
}

func TestTorrentSearch_DefaultPagination(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		assert.InDelta(t, 0, reqBody["from"], 0.0001)
		assert.InEpsilon(t, 10, reqBody["size"], 0.0001)

		testutil.ESOK(w, `{
			"took": 1,
			"timed_out": false,
			"hits": {
				"total": {"value":0,"relation":"eq"},
				"hits": []
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	_, err = s.TorrentSearch(context.Background(), search.TorrentSearchParams{})
	require.NoError(t, err)
}

func TestTorrentSearch_HasNextPage(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, _ *http.Request) {
		testutil.ESOK(w, `{
			"took": 1,
			"timed_out": false,
			"hits": {
				"total": {"value":25,"relation":"eq"},
				"hits": [
					{"_index":"torrent_content","_id":"1","_score":1.0,"_source":{
						"info_hash":"a1","name":"t1","size":1,"created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"
					}},
					{"_index":"torrent_content","_id":"2","_score":1.0,"_source":{
						"info_hash":"a2","name":"t2","size":1,"created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"
					}}
				]
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	result, err := s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		Offset: 20,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.True(t, result.HasNextPage)
}

func TestTorrentSearch_GTERelation(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, _ *http.Request) {
		testutil.ESOK(w, `{
			"took": 1,
			"timed_out": false,
			"hits": {
				"total": {"value":10000,"relation":"gte"},
				"hits": []
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	result, err := s.TorrentSearch(context.Background(), search.TorrentSearchParams{})
	require.NoError(t, err)
	assert.True(t, result.TotalCountIsEstimate)
	assert.Equal(t, uint(10000), result.TotalCount)
}

func TestTorrentSearch_NullSeedersLeechers(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, _ *http.Request) {
		testutil.ESOK(w, `{
			"took": 1,
			"timed_out": false,
			"hits": {
				"total": {"value":1,"relation":"eq"},
				"hits": [
					{"_index":"torrent_content","_id":"1","_score":1.0,"_source":{
						"info_hash":"abc","name":"test","size":1,"seeders":null,"leechers":null,
						"files_count":null,"created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"
					}}
				]
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	result, err := s.TorrentSearch(context.Background(), search.TorrentSearchParams{})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Nil(t, result.Items[0].Seeders)
	assert.Nil(t, result.Items[0].Leechers)
	assert.Nil(t, result.Items[0].FilesCount)
}

func TestTorrentSearch_UnknownContentType(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, _ *http.Request) {
		testutil.ESOK(w, `{
			"took": 1,
			"timed_out": false,
			"hits": {
				"total": {"value":1,"relation":"eq"},
				"hits": [
					{"_index":"torrent_content","_id":"1","_score":1.0,"_source":{
						"info_hash":"abc","name":"test","size":1,"content_type":"unknown",
						"created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"
					}}
				]
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	result, err := s.TorrentSearch(context.Background(), search.TorrentSearchParams{})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Nil(t, result.Items[0].ContentType)
}

func TestTorrentSearch_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	_, err = s.TorrentSearch(context.Background(), search.TorrentSearchParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "es search")
}

func TestEsSortField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		field search.TorrentSearchField
		want  string
	}{
		{search.FieldCreatedAt, "created_at"},
		{search.FieldUpdatedAt, "updated_at"},
		{search.FieldSize, "size"},
		{search.FieldFilesCount, "files_count"},
		{search.FieldSeeders, "seeders"},
		{search.FieldLeechers, "leechers"},
		{search.FieldName, "name.keyword"},
		{search.FieldInfoHash, "info_hash"},
		{search.FieldRelevance, "_score"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.field), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, esSortField(tt.field))
		})
	}
}

func TestHasFacets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  search.FacetAggregationConfig
		want bool
	}{
		{"all false", search.FacetAggregationConfig{}, false},
		{"content type", search.FacetAggregationConfig{ContentType: true}, true},
		{"torrent source", search.FacetAggregationConfig{TorrentSource: true}, true},
		{"file type", search.FacetAggregationConfig{FileType: true}, true},
		{"language", search.FacetAggregationConfig{Language: true}, true},
		{"release year", search.FacetAggregationConfig{ReleaseYear: true}, true},
		{"multiple", search.FacetAggregationConfig{ContentType: true, Language: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, hasFacets(tt.cfg))
		})
	}
}

func TestBuildAggs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  search.TorrentSearchParams
		want []string
	}{
		{
			"content type",
			search.TorrentSearchParams{FacetAggregate: search.FacetAggregationConfig{ContentType: true}},
			[]string{"content_type"},
		},
		{
			"torrent source",
			search.TorrentSearchParams{FacetAggregate: search.FacetAggregationConfig{TorrentSource: true}},
			[]string{"torrent_source"},
		},
		{
			"file type",
			search.TorrentSearchParams{FacetAggregate: search.FacetAggregationConfig{FileType: true}},
			[]string{"file_type"},
		},
		{
			"language",
			search.TorrentSearchParams{FacetAggregate: search.FacetAggregationConfig{Language: true}},
			[]string{"language"},
		},
		{
			"release year",
			search.TorrentSearchParams{FacetAggregate: search.FacetAggregationConfig{ReleaseYear: true}},
			[]string{"release_year"},
		},
		{
			"multiple",
			search.TorrentSearchParams{FacetAggregate: search.FacetAggregationConfig{ContentType: true, Language: true}},
			[]string{"content_type", "language"},
		},
		{
			"none",
			search.TorrentSearchParams{},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			aggs := buildAggs(tt.cfg)
			if tt.want == nil {
				assert.Empty(t, aggs)
			} else {
				for _, key := range tt.want {
					assert.Contains(t, aggs, key)
				}

				assert.Len(t, aggs, len(tt.want))
			}
		})
	}
}

func TestParseAggs(t *testing.T) {
	t.Parallel()

	raw := map[string]json.RawMessage{
		"content_type": json.RawMessage(`{
			"doc_count_error_upper_bound": 0,
			"sum_other_doc_count": 0,
			"buckets": [
				{"key": "movie", "doc_count": 10},
				{"key": "tv", "doc_count": 5}
			]
		}`),
		"language": json.RawMessage(`{
			"buckets": [
				{"key": "en", "doc_count": 8}
			]
		}`),
	}

	aggs, err := parseAggs(raw, search.TorrentSearchParams{
		FacetAggregate: search.FacetAggregationConfig{
			ContentType: true,
			Language:    true,
		},
	})
	require.NoError(t, err)

	require.Contains(t, aggs, "content_type")
	assert.Equal(t, uint(10), aggs["content_type"].Items["movie"].Count)
	assert.Equal(t, uint(5), aggs["content_type"].Items["tv"].Count)

	require.Contains(t, aggs, "language")
	assert.Equal(t, uint(8), aggs["language"].Items["en"].Count)
}

func TestParseAggs_DisabledFacets(t *testing.T) {
	t.Parallel()

	raw := map[string]json.RawMessage{
		"content_type": json.RawMessage(`{"buckets":[{"key":"movie","doc_count":5}]}`),
	}
	aggs, err := parseAggs(raw, search.TorrentSearchParams{})
	require.NoError(t, err)
	assert.Empty(t, aggs)
}

func TestParseAggs_MissingBucket(t *testing.T) {
	t.Parallel()

	raw := map[string]json.RawMessage{
		"content_type": json.RawMessage(`{"buckets":[{"key":"movie","doc_count":5}]}`),
	}
	aggs, err := parseAggs(raw, search.TorrentSearchParams{
		FacetAggregate: search.FacetAggregationConfig{
			Language: true,
		},
	})
	require.NoError(t, err)
	assert.Empty(t, aggs)
}

func TestParseAggs_MalformedJSON(t *testing.T) {
	t.Parallel()

	raw := map[string]json.RawMessage{
		"content_type": json.RawMessage(`invalid json`),
	}
	aggs, err := parseAggs(raw, search.TorrentSearchParams{
		FacetAggregate: search.FacetAggregationConfig{ContentType: true},
	})
	require.NoError(t, err)
	assert.Empty(t, aggs["content_type"].Items)
}

func TestHitToRow(t *testing.T) {
	t.Parallel()

	src := json.RawMessage(`{
		"info_hash":"abc123",
		"name":"test torrent",
		"size":1024,
		"seeders":10,
		"leechers":5,
		"files_count":3,
		"private":true,
		"content_type":"movie",
		"content_source":"tmdb",
		"content_id":"123",
		"title":"A Movie",
		"overview":"Great movie",
		"languages":["en","fr"],
		"sources":["public"],
				"created_at":"2024-01-15T10:00:00Z",
		"updated_at":"2024-01-16T10:00:00Z"
	}`)

	row, err := hitToRow(src)
	require.NoError(t, err)

	assert.Equal(t, "abc123", row.InfoHash)
	assert.Equal(t, "test torrent", row.TorrentName)
	assert.Equal(t, int64(1024), row.Size)
	assert.True(t, row.TorrentPrivate)

	require.NotNil(t, row.Seeders)
	assert.Equal(t, int32(10), *row.Seeders)
	require.NotNil(t, row.Leechers)
	assert.Equal(t, int32(5), *row.Leechers)
	require.NotNil(t, row.FilesCount)
	assert.Equal(t, int32(3), *row.FilesCount)

	require.NotNil(t, row.ContentType)
	assert.Equal(t, "movie", *row.ContentType)
	require.NotNil(t, row.ContentSource)
	assert.Equal(t, "tmdb", *row.ContentSource)
	require.NotNil(t, row.ContentID)
	assert.Equal(t, "123", *row.ContentID)
	require.NotNil(t, row.ContentTitle)
	assert.Equal(t, "A Movie", *row.ContentTitle)
	require.NotNil(t, row.ContentOverview)
	assert.Equal(t, "Great movie", *row.ContentOverview)

	assert.Equal(t, 2024, row.CreatedAt.Year())
	assert.Equal(t, 2024, row.UpdatedAt.Year())
}

func TestHitToRow_Minimal(t *testing.T) {
	t.Parallel()

	src := json.RawMessage(`{
		"info_hash":"abc",
		"name":"test",
		"size":0,
				"created_at":"2024-01-15T10:00:00Z",
		"updated_at":"2024-01-15T10:00:00Z"
	}`)

	row, err := hitToRow(src)
	require.NoError(t, err)
	assert.Equal(t, "abc", row.InfoHash)
	assert.Nil(t, row.Seeders)
	assert.Nil(t, row.Leechers)
	assert.Nil(t, row.FilesCount)
	assert.Nil(t, row.ContentType)
	assert.Nil(t, row.ContentSource)
	assert.Nil(t, row.ContentID)
	assert.Nil(t, row.ContentTitle)
	assert.Nil(t, row.ContentOverview)
	assert.Nil(t, row.Languages)
}

func TestHitToRow_Error(t *testing.T) {
	t.Parallel()

	_, err := hitToRow(json.RawMessage(`invalid json`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestParseISO(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  time.Time
	}{
		{"2024-01-15T10:00:00Z", time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)},
		{"2024-01-15T10:00:00+00:00", time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)},
		{"2024-01-15T10:00:00+02:00", time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)},
		{"invalid", time.Time{}},
		{"", time.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := parseISO(tt.input)
			assert.True(t, got.Equal(tt.want), "parseISO(%q) = %v, want %v", tt.input, got, tt.want)
		})
	}
}

func TestParseISOAltFormat(t *testing.T) {
	t.Parallel()

	got := parseISO("2024-01-15T10:00:05Z")
	assert.Equal(t, 5, got.Second())
}

func TestStringSliceToBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input []string
		want  string
	}{
		{[]string{"en", "fr"}, `["en","fr"]`},
		{[]string{"en"}, `["en"]`},
		{nil, ""},
		{[]string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			got := stringSliceToBytes(tt.input)
			if tt.want == "" {
				assert.Nil(t, got)
			} else {
				assert.JSONEq(t, tt.want, string(got))
			}
		})
	}
}

func TestToAnySlice(t *testing.T) {
	t.Parallel()

	ints := []int32{1, 2, 3}
	got := toAnySlice(ints)
	assert.Equal(t, []any{int32(1), int32(2), int32(3)}, got)

	strs := []string{"a", "b"}
	got2 := toAnySlice(strs)
	assert.Equal(t, []any{"a", "b"}, got2)

	got3 := toAnySlice([]int32{})
	assert.Empty(t, got3)
}

func TestBuildSimpleAgg(t *testing.T) {
	t.Parallel()

	agg := buildSimpleAgg("content_type")
	terms, ok := agg["terms"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "content_type", terms["field"])
	assert.Equal(t, 100, terms["size"])
}

func makeHit(id string) elasticsearch.SearchHit {
	return elasticsearch.SearchHit{
		Index: "torrent_content",
		ID:    id,
		Score: float64Ptr(1.0),
		Source: json.RawMessage(
			`{"info_hash":"` + id + `","name":"test","size":1,"created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}`,
		),
	}
}

func float64Ptr(v float64) *float64 { return &v }

func TestRRFMerge_BM25Only(t *testing.T) {
	t.Parallel()

	bm25 := []elasticsearch.SearchHit{makeHit("a"), makeHit("b")}

	var knn []elasticsearch.SearchHit

	res := rrfMerge(bm25, knn, 10, 10, 0, nil)
	require.Len(t, res.hits, 2)
	assert.Equal(t, "a", res.hits[0].ID)
	assert.Equal(t, "b", res.hits[1].ID)
	assert.Equal(t, 2, res.total)
}

func TestRRFMerge_kNNOnly(t *testing.T) {
	t.Parallel()

	var bm25 []elasticsearch.SearchHit

	knn := []elasticsearch.SearchHit{makeHit("a"), makeHit("b")}

	res := rrfMerge(bm25, knn, 10, 10, 0, nil)
	require.Len(t, res.hits, 2)
	assert.Equal(t, "a", res.hits[0].ID)
	assert.Equal(t, "b", res.hits[1].ID)
	assert.Equal(t, 2, res.total)
}

func TestRRFMerge_Mixed(t *testing.T) {
	t.Parallel()

	bm25 := []elasticsearch.SearchHit{makeHit("a"), makeHit("b"), makeHit("c")}
	knn := []elasticsearch.SearchHit{makeHit("d"), makeHit("e")}

	res := rrfMerge(bm25, knn, 10, 10, 0, nil)
	require.Len(t, res.hits, 5)
	assert.Equal(t, 5, res.total)
}

func TestRRFMerge_kNNResultsIncluded(t *testing.T) {
	t.Parallel()

	bm25 := []elasticsearch.SearchHit{makeHit("only_bm25")}
	knn := []elasticsearch.SearchHit{makeHit("only_knn")}

	res := rrfMerge(bm25, knn, 10, 10, 0, nil)
	require.Len(t, res.hits, 2)

	ids := map[string]bool{}
	for _, h := range res.hits {
		ids[h.ID] = true
	}

	assert.True(t, ids["only_bm25"], "BM25 result should be included")
	assert.True(t, ids["only_knn"], "kNN result should be included")
	assert.Equal(t, 2, res.total)
}

func TestRRFMerge_Deduplication(t *testing.T) {
	t.Parallel()

	hit := makeHit("dup")
	bm25 := []elasticsearch.SearchHit{hit}
	knn := []elasticsearch.SearchHit{hit}

	res := rrfMerge(bm25, knn, 10, 10, 0, nil)
	require.Len(t, res.hits, 1, "duplicate ID should appear only once")
	assert.Equal(t, "dup", res.hits[0].ID)
	assert.Equal(t, 1, res.total)
}

func TestRRFMerge_BM25HigherWeight(t *testing.T) {
	t.Parallel()

	bm25 := []elasticsearch.SearchHit{makeHit("bm25_first"), makeHit("bm25_second")}
	knn := []elasticsearch.SearchHit{makeHit("knn_first"), makeHit("knn_second")}

	// BM25 has lower k constant (30 vs 60), so BM25 rank 1 > kNN rank 1:
	//   BM25 rank 1: 3/(1+30) = 3/31 ≈ 0.097
	//   kNN rank 1: 3/(1+60) = 3/61 ≈ 0.049
	//   BM25 rank 2: 3/(1+31) = 3/32 ≈ 0.094
	res := rrfMerge(bm25, knn, 10, 10, 0, nil)
	require.Len(t, res.hits, 4)
	assert.Equal(t, "bm25_first", res.hits[0].ID, "BM25 rank 1 should outrank kNN rank 1")
	assert.Equal(t, "bm25_second", res.hits[1].ID, "BM25 rank 2 should outrank kNN rank 1")
	assert.Equal(t, 4, res.total)
}

func TestRRFMerge_Pagination(t *testing.T) {
	t.Parallel()

	bm25 := []elasticsearch.SearchHit{makeHit("a"), makeHit("b"), makeHit("c")}

	var knn []elasticsearch.SearchHit

	res := rrfMerge(bm25, knn, 10, 1, 1, nil)
	require.Len(t, res.hits, 1)
	assert.Equal(t, "b", res.hits[0].ID, "offset 1, limit 1 should return second item")
	assert.Equal(t, 3, res.total)
}

func TestRRFMerge_PaginationOffsetExceedsResults(t *testing.T) {
	t.Parallel()

	bm25 := []elasticsearch.SearchHit{makeHit("a")}

	var knn []elasticsearch.SearchHit

	res := rrfMerge(bm25, knn, 10, 10, 5, nil)
	assert.Empty(t, res.hits, "offset beyond results should return empty")
	assert.Equal(t, 1, res.total)
}

func TestRRFMerge_EmptyInputs(t *testing.T) {
	t.Parallel()

	res := rrfMerge(nil, nil, 10, 10, 0, nil)
	assert.Empty(t, res.hits)
	assert.Equal(t, 0, res.total)
}

func TestRRFMerge_WindowSizeLimit(t *testing.T) {
	t.Parallel()

	bm25 := []elasticsearch.SearchHit{
		makeHit("a"), makeHit("b"), makeHit("c"),
		makeHit("d"), makeHit("e"),
	}

	var knn []elasticsearch.SearchHit

	res := rrfMerge(bm25, knn, 3, 10, 0, nil)
	require.Len(t, res.hits, 3, "windowSize 3 limits results")
	assert.Equal(t, 3, res.total)
}

func TestTorrentSearch_ContentTypeFilter(t *testing.T) {
	t.Parallel()

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		// content_type should be in filter clauses, NOT in post_filter
		assert.NotContains(t, reqBody, "post_filter")

		query, ok := reqBody["query"].(map[string]any)
		assert.True(t, ok)
		boolQ, ok := query["bool"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, boolQ, "filter")

		// Find the content_type filter in filter clauses
		filters, ok := boolQ["filter"].([]any)
		assert.True(t, ok)

		found := false

		for _, f := range filters {
			fmap, ok := f.(map[string]any)
			if !ok {
				continue
			}

			terms, ok := fmap["terms"].(map[string]any)
			if !ok {
				continue
			}

			if ct, ok := terms["content_type"]; ok {
				found = true

				assert.Equal(t, []any{"music"}, ct)
			}
		}

		assert.True(t, found, "content_type filter should be in filter clauses")

		testutil.ESOK(w, `{
			"took": 1, "timed_out": false,
			"hits": {"total": {"value": 2, "relation": "eq"}, "hits": [
				{"_id": "1", "_score": 1.0, "_source": {"info_hash":"a","name":"t1","size":1,
					"content_type":"music","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}},
				{"_id": "2", "_score": 1.0, "_source": {"info_hash":"b","name":"t2","size":1,
					"content_type":"music","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}}
			]},
			"aggregations": {
				"content_type": {"doc_count_error_upper_bound": 0, "sum_other_doc_count": 0, "buckets": [
					{"key": "music", "doc_count": 2}
				]}
			}
		}`)
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	s := New(client, nil, nil, nil, false)
	result, err := s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		ContentTypes: []string{"music"},
	})
	require.NoError(t, err)
	assert.Len(t, result.Items, 2)
}

func TestTorrentSearch_kNNScoreFiltering_KeepsHighScore(t *testing.T) {
	t.Parallel()

	var requestCount int

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		requestCount++
		switch requestCount {
		case 1:
			// BM25 RRF sub-query
			testutil.ESOK(w, `{
				"took": 1, "timed_out": false,
				"hits": {"total": {"value": 3, "relation": "eq"}, "hits": [
					{"_id": "b1", "_score": 1.0, "_source": {"info_hash":"b1","name":"bm25","size":1,
						"content_type":"movie","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}}
				]},
				"aggregations": {
					"content_type": {"doc_count_error_upper_bound": 0, "sum_other_doc_count": 0, "buckets": [
						{"key": "movie", "doc_count": 1}
					]}
				}
			}`)
		case 2:
			// kNN sub-query — mix of high and low scores
			testutil.ESOK(w, `{
				"took": 1, "timed_out": false,
				"hits": {"total": {"value": 5, "relation": "eq"}, "hits": [
					{"_id": "k1", "_score": 0.85, "_source": {"info_hash":"k1","name":"high1","size":1,
						"content_type":"music","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}},
					{"_id": "k2", "_score": 0.80, "_source": {"info_hash":"k2","name":"high2","size":1,
						"content_type":"movie","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}},
					{"_id": "k3", "_score": 0.30, "_source": {"info_hash":"k3","name":"low1","size":1,
						"content_type":"music","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}},
					{"_id": "k4", "_score": 0.25, "_source": {"info_hash":"k4","name":"low2","size":1,
						"content_type":"music","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}}
				]}
			}`)
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	mockEmb := &mockEmbedder{
		embedFunc: func(_ context.Context, _ []string) ([][]float32, error) {
			return [][]float32{{0.1, 0.2, 0.3}}, nil
		},
	}

	s := New(client, nil, mockEmb, nil, false)
	result, err := s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		QueryString: "test",
		Limit:       10,
	})
	require.NoError(t, err)

	// topScore = 0.85, threshold = max(0.85*0.6, 0.3) = 0.51
	// k1 (0.85) and k2 (0.80) should be kept; k3 (0.30) and k4 (0.25) filtered out
	// BM25: 1 item + kNN: 2 items = 3 items
	require.Len(t, result.Items, 3)

	// content_type bucket should reflect filtered kNN + BM25
	itemIDs := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		itemIDs = append(itemIDs, item.InfoHash)
	}

	assert.Contains(t, itemIDs, "b1")
	assert.Contains(t, itemIDs, "k1")
	assert.Contains(t, itemIDs, "k2")
	assert.NotContains(t, itemIDs, "k3")
	assert.NotContains(t, itemIDs, "k4")
}

func TestTorrentSearch_kNNScoreFiltering_DegradeOnLowTopScore(t *testing.T) {
	t.Parallel()

	var (
		requestCount int
		fallbackBody map[string]any
	)

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		requestCount++
		switch requestCount {
		case 1:
			// BM25 RRF sub-query
			testutil.ESOK(w, `{
				"took": 1, "timed_out": false,
				"hits": {"total": {"value": 2, "relation": "eq"}, "hits": [
					{"_id": "b1", "_score": 1.0, "_source": {"info_hash":"b1","name":"bm25","size":1,
						"content_type":"movie","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}}
				]},
				"aggregations": {
					"content_type": {"doc_count_error_upper_bound": 0, "sum_other_doc_count": 0, "buckets": [
						{"key": "movie", "doc_count": 2}
					]}
				}
			}`)
		case 2:
			// kNN sub-query — all scores below 0.2
			testutil.ESOK(w, `{
				"took": 1, "timed_out": false,
				"hits": {"total": {"value": 2, "relation": "eq"}, "hits": [
					{"_id": "k1", "_score": 0.15, "_source": {"info_hash":"k1","name":"low1","size":1,
						"content_type":"music","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}},
					{"_id": "k2", "_score": 0.12, "_source": {"info_hash":"k2","name":"low2","size":1,
						"content_type":"music","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}}
				]}
			}`)
		case 3:
			// BM25 fallback — should have pagination restored
			fallbackBody = reqBody

			testutil.ESOK(w, `{
				"took": 1, "timed_out": false,
				"hits": {"total": {"value": 2, "relation": "eq"}, "hits": [
					{"_id": "b1", "_score": 1.0, "_source": {"info_hash":"b1","name":"bm25","size":1,
						"content_type":"movie","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}}
				]},
				"aggregations": {
					"content_type": {"doc_count_error_upper_bound": 0, "sum_other_doc_count": 0, "buckets": [
						{"key": "movie", "doc_count": 2}
					]}
				}
			}`)
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	mockEmb := &mockEmbedder{
		embedFunc: func(_ context.Context, _ []string) ([][]float32, error) {
			return [][]float32{{0.1, 0.2, 0.3}}, nil
		},
	}

	s := New(client, nil, mockEmb, nil, false)
	result, err := s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		QueryString: "test",
		Limit:       10,
	})
	require.NoError(t, err)

	// kNN degraded to BM25-only, so should have 1 BM25 result
	require.Len(t, result.Items, 1)
	assert.Equal(t, "b1", result.Items[0].InfoHash)
	// Should fall back to BM25 with pagination restored
	require.NotNil(t, fallbackBody)
}

func TestTorrentSearch_FallbackPaginationRestored(t *testing.T) {
	t.Parallel()

	var (
		requestCount               int
		fallbackFrom, fallbackSize any
	)

	srv := newTestESSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any

		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		requestCount++
		switch requestCount {
		case 1:
			// BM25 RRF search — should use from=0, size=200
			assert.InDelta(t, 0, reqBody["from"], 0.0001)
			assert.InEpsilon(t, 200, reqBody["size"], 0.0001)
			testutil.ESOK(w, `{
				"took": 1, "timed_out": false,
				"hits": {"total": {"value": 5, "relation": "eq"}, "hits": [
					{"_id": "a", "_score": 1.0, "_source": {"info_hash": "a", "name": "hit_a"}}
				]}
			}`)
		case 2:
			// kNN search — fail to trigger fallback
			w.Header().Set("X-Elastic-Product", "Elasticsearch")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"knn failed"}`))
		case 3:
			// BM25 fallback — should have ORIGINAL from/size restored
			fallbackFrom = reqBody["from"]
			fallbackSize = reqBody["size"]

			testutil.ESOK(w, `{
				"took": 1, "timed_out": false,
				"hits": {"total": {"value": 5, "relation": "eq"}, "hits": [
					{"_id": "a", "_score": 1.0, "_source": {"info_hash": "a", "name": "hit_a"}}
				]}
			}`)
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	})
	defer srv.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	mockEmb := &mockEmbedder{
		embedFunc: func(_ context.Context, _ []string) ([][]float32, error) {
			return [][]float32{{0.1, 0.2, 0.3}}, nil
		},
	}

	s := New(client, nil, mockEmb, nil, false)
	_, err = s.TorrentSearch(context.Background(), search.TorrentSearchParams{
		QueryString: "test",
		Limit:       10,
		Offset:      2,
		TotalCount:  true,
	})
	require.NoError(t, err)
	assert.InEpsilon(t, 2, fallbackFrom, 0.0001, "original offset 2 should be restored")
	assert.InEpsilon(t, 10, fallbackSize, 0.0001, "original limit 10 should be restored")
}

func newTestESSearchServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && r.Method == http.MethodGet {
			w.Header().Set("X-Elastic-Product", "Elasticsearch")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name":"test"}`))

			return
		}

		handler(w, r)
	}))
}
