package classifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStrictLLM_WorkflowAbortsOnLLMFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "llm unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	mocks := newTestClassifierMocks(t)
	mocks.compiler.dependencies.logger = zap.NewNop().Sugar()
	mocks.compiler.defaultLLMConfig = LLMConfig{
		Endpoint: server.URL,
		APIKey:   "test-key",
		Model:    "test-model",
		Enabled:  true,
	}

	source, err := yamlSourceProvider{rawSourceProvider: coreSourceProvider{}}.source()
	require.NoError(t, err)

	workflow, err := mocks.compiler.Compile(source)
	require.NoError(t, err)

	torrent := model.Torrent{Name: "The Regular Show S01E01 1080p WEBRip.mkv"}

	flags := Flags{
		"local_search_enabled": false,
		"apis_enabled":         false,
		"tmdb_enabled":         false,
		"llm_enabled":          true,
	}

	_, strictErr := workflow.Run(WithStrictLLM(context.Background()), "default", flags, torrent)
	require.Error(t, strictErr)

	var llmErr *LLMClassifyError
	require.ErrorAs(t, strictErr, &llmErr)

	_, fallbackErr := workflow.Run(context.Background(), "default", flags, torrent)
	assert.NoError(t, fallbackErr, "non-strict classification must fall back to rules")
}

func TestStrictLLM_ContextHelpers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	assert.False(t, StrictLLM(ctx))
	assert.True(t, StrictLLM(WithStrictLLM(ctx)))

	err := &LLMClassifyError{Cause: assert.AnError}
	require.ErrorIs(t, err, assert.AnError)
	assert.Contains(t, err.Error(), "llm classify failed")
}
