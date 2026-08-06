package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		testutil.ESOK(w, `{}`)
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestPing(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		testutil.ESOK(w, `{"name":"es-node-1"}`)
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	err = client.Ping(context.Background())
	assert.NoError(t, err)
}

func TestPing_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"cluster_unavailable"}`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	err = client.Ping(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping")
}

func TestCreateIndex(t *testing.T) {
	t.Parallel()

	var (
		gotPath, gotMethod string
		gotBody            bytes.Buffer
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = gotBody.ReadFrom(r.Body)

		testutil.ESOK(w, `{"acknowledged":true,"shards_acknowledged":true,"index":"test_index"}`)
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	mapping := strings.NewReader(`{"settings":{"number_of_shards":1}}`)
	err = client.CreateIndex(context.Background(), "test_index", mapping)
	require.NoError(t, err)
	assert.Equal(t, "PUT", gotMethod)
	assert.Equal(t, "/test_index", gotPath)
	assert.JSONEq(t, `{"settings":{"number_of_shards":1}}`, gotBody.String())
}

func TestCreateIndex_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"type":"index_already_exists_exception","reason":"already exists"}}`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	err = client.CreateIndex(context.Background(), "test_index", strings.NewReader(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create index")
}

func TestIndexExists(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "HEAD", r.Method)
		assert.Equal(t, "/test_index", r.URL.Path)
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	exists, err := client.IndexExists(context.Background(), "test_index")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestIndexExists_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	exists, err := client.IndexExists(context.Background(), "test_index")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestIndexExists_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	_, err = client.IndexExists(context.Background(), "test_index")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check index exists")
}

func TestBulkIndex(t *testing.T) {
	t.Parallel()

	var gotBody bytes.Buffer

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/_bulk", r.URL.Path)
		_, _ = gotBody.ReadFrom(r.Body)

		testutil.ESOK(w, `{"errors":false,"items":[{"index":{"_id":"1","status":201}}]}`)
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	body := strings.NewReader(`{"index":{"_index":"test","_id":"1"}}\n{"title":"doc1"}\n`)
	err = client.BulkIndex(context.Background(), body)
	require.NoError(t, err)
	assert.NotEmpty(t, gotBody.String())
}

func TestBulkIndex_ESError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad_request"}`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	err = client.BulkIndex(context.Background(), strings.NewReader(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bulk index")
}

func TestBulkIndex_DocErrors(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		testutil.ESOK(
			w,
			`{"errors":true,"items":[{"index":{"_id":"1","status":400,"error":{"type":"mapper_parsing_exception",`+
				`"reason":"failed to parse"}}}]}`,
		)
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	err = client.BulkIndex(context.Background(), strings.NewReader(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bulk index")
	assert.Contains(t, err.Error(), "mapper_parsing_exception")
}

func TestBulkIndex_DecodeError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	err = client.BulkIndex(context.Background(), strings.NewReader(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestDeleteIndex(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		testutil.ESOK(w, `{"acknowledged":true}`)
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	err = client.DeleteIndex(context.Background(), "test_index")
	require.NoError(t, err)
	assert.Equal(t, "DELETE", gotMethod)
	assert.Equal(t, "/test_index", gotPath)
}

func TestDeleteIndex_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"index_not_found"}`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	err = client.DeleteIndex(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete index")
}

func TestSearch(t *testing.T) {
	t.Parallel()

	var gotBody bytes.Buffer

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/test_index/_search", r.URL.Path)
		_, _ = gotBody.ReadFrom(r.Body)

		testutil.ESOK(w, `{
			"took": 5,
			"timed_out": false,
			"_shards": {"total":1,"successful":1,"failed":0},
			"hits": {
				"total": {"value":2,"relation":"eq"},
				"max_score": 1.0,
				"hits": [
					{"_index":"test_index","_id":"1","_score":1.0,"_source":{"info_hash":"abc123","name":"test torrent",`+
			`"size":1024,"created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}},
					{"_index":"test_index","_id":"2","_score":0.5,"_source":{"info_hash":"def456","name":"another torrent",`+
			`"size":2048,"content_type":"movie","content_source":"tmdb","content_id":"123","title":"A Movie",`+
			`"overview":"Great movie","seeders":10,"leechers":5,"files_count":3,"languages":["en","fr"],`+
			`"sources":["public"],"private":false,"created_at":"2024-01-14T10:00:00Z",`+
			`"updated_at":"2024-01-14T10:00:00Z"}}
				]
			},
			"aggregations": {
				"content_type": {"doc_count_error_upper_bound":0,"sum_other_doc_count":0,"buckets":[{"key":"movie","doc_count":1}]}
			}
		}`)
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	body := map[string]any{
		"query": map[string]any{
			"match": map[string]any{"name": "test"},
		},
	}
	result, err := client.Search(context.Background(), "test_index", body)
	require.NoError(t, err)

	assert.Equal(t, 5, result.Took)
	assert.False(t, result.TimedOut)
	assert.Equal(t, int64(2), result.Hits.Total.Value)
	assert.Equal(t, "eq", result.Hits.Total.Relation)
	assert.Len(t, result.Hits.Hits, 2)
	assert.Len(t, result.Aggs, 1)

	var searchBody map[string]any

	err = json.Unmarshal(gotBody.Bytes(), &searchBody)
	require.NoError(t, err)

	query, ok := searchBody["query"].(map[string]any)
	require.True(t, ok)
	match, ok := query["match"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test", match["name"])
}

func TestSearch_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"root_cause":[{"type":"query_shard_exception","reason":"failed to parse"}]}}`))
	}))
	defer srv.Close()

	client, err := NewClient(Config{Addresses: []string{srv.URL}})
	require.NoError(t, err)

	_, err = client.Search(context.Background(), "test", map[string]any{"invalid": "query"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search")
}
