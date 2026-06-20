package tmdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequester_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test/path", r.URL.Path)
		assert.Equal(t, "value1", r.URL.Query().Get("key1"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "ok"}`))
	}))
	defer srv.Close()

	r := requester{
		resty: resty.New().SetBaseURL(srv.URL),
	}
	resp, err := r.Request(context.Background(), "/test/path", map[string]string{"key1": "value1"}, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.IsSuccess())
	assert.Contains(t, resp.String(), `"result": "ok"`)
}

func TestRequester_NoQueryParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/no-params", r.URL.Path)
		assert.Empty(t, r.URL.Query())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	r := requester{
		resty: resty.New().SetBaseURL(srv.URL),
	}
	_, err := r.Request(context.Background(), "/no-params", nil, nil)
	require.NoError(t, err)
}

func TestRequester_Unauthorized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	r := requester{
		resty: resty.New().SetBaseURL(srv.URL),
	}
	_, err := r.Request(context.Background(), "/test", nil, nil)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestRequester_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := requester{
		resty: resty.New().SetBaseURL(srv.URL),
	}
	_, err := r.Request(context.Background(), "/test", nil, nil)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRequester_InternalServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	r := requester{
		resty: resty.New().SetBaseURL(srv.URL),
	}
	_, err := r.Request(context.Background(), "/test", nil, nil)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrUnauthorized)
	require.NotErrorIs(t, err, ErrNotFound)
}

func TestRequester_BadRequest(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	r := requester{
		resty: resty.New().SetBaseURL(srv.URL),
	}
	_, err := r.Request(context.Background(), "/test", nil, nil)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrUnauthorized)
	require.NotErrorIs(t, err, ErrNotFound)
}

func TestRequester_Forbidden(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	r := requester{
		resty: resty.New().SetBaseURL(srv.URL),
	}
	_, err := r.Request(context.Background(), "/test", nil, nil)
	assert.Error(t, err)
}

func TestRequester_WithResultPtr(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"page": 1, "total_results": 10, "total_pages": 1, "results": []}`))
	}))
	defer srv.Close()

	r := requester{
		resty: resty.New().SetBaseURL(srv.URL),
	}
	resp, err := r.Request(context.Background(), "/test", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, resp.String(), `"page": 1`)
	assert.Contains(t, resp.String(), `"total_results": 10`)
}
