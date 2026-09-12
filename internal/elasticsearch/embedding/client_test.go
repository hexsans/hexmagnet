package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hexsans/hexmagnet/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbedSendsUserAgent(t *testing.T) {
	t.Parallel()

	var gotUA atomic.Pointer[string]

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Header.Get("User-Agent")
		gotUA.Store(&v)

		_ = json.NewEncoder(w).Encode(embedResponse{
			Data: []embedDataItem{{Embedding: []float32{1, 2}}},
		})
	}))
	defer srv.Close()

	client := NewClient(Config{Endpoint: srv.URL, Model: "test-model"})

	_, err := client.Embed(context.Background(), []string{"hello"})
	require.NoError(t, err)

	require.NotNil(t, gotUA.Load())
	assert.Equal(t, version.UserAgent(), *gotUA.Load())
}
