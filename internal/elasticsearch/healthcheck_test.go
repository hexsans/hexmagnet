package elasticsearch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHealthCheck(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"test"}`))
	}))
	defer srv.Close()

	check := NewHealthCheck(
		func() bool { return true },
		func() (*Client, error) {
			return NewClient(Config{Addresses: []string{srv.URL}})
		},
	)
	assert.Equal(t, "elasticsearch", check.Name)
	assert.Equal(t, 10*time.Second, check.Timeout)
	assert.True(t, check.IsActive())

	err := check.Check(context.Background())
	assert.NoError(t, err)
}

func TestNewHealthCheck_Inactive(t *testing.T) {
	t.Parallel()

	check := NewHealthCheck(
		func() bool { return false },
		func() (*Client, error) {
			return NewClient(Config{Addresses: []string{defaultAddress}})
		},
	)

	assert.False(t, check.IsActive())
}

func TestNewHealthCheck_Failure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	check := NewHealthCheck(
		func() bool { return true },
		func() (*Client, error) {
			return NewClient(Config{Addresses: []string{srv.URL}})
		},
	)

	err := check.Check(context.Background())
	assert.Error(t, err)
}

func TestNewHealthCheck_ClientError(t *testing.T) {
	t.Parallel()

	expected := errors.New("client error")
	check := NewHealthCheck(
		func() bool { return true },
		func() (*Client, error) {
			return nil, expected
		},
	)

	err := check.Check(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, expected)
}
