package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const (
	testMovieType = "movie"
	testModel     = "gpt-4o-mini"
)

func llmResultJSON(t *testing.T, result LLMResult) string {
	t.Helper()

	b, err := json.Marshal(result)
	require.NoError(t, err)

	return string(b)
}

func chatResponseBody(t *testing.T, content string) string {
	t.Helper()

	b, err := json.Marshal(map[string]any{
		"choices": []map[string]any{{
			"message": map[string]any{
				"content": content,
			},
		}},
	})
	require.NoError(t, err)

	return string(b)
}

func TestClassify_SuccessFirstAttempt(t *testing.T) {
	t.Parallel()

	var callCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
		}

		expected := LLMResult{
			Type:      testMovieType,
			BaseTitle: "Test Movie",
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponseBody(t, llmResultJSON(t, expected))))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 3,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	result, err := c.Classify(context.Background(), "Test Movie 2024", []TorrentFile{}, "testhash")
	require.NoError(t, err)
	require.Equal(t, 1, callCount)
	require.Equal(t, testMovieType, result.Type)
	require.Equal(t, "Test Movie", result.BaseTitle)
}

func TestClassify_SuccessOnRetry(t *testing.T) {
	t.Parallel()

	var callCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": {"message": "server error"}}`))

			return
		}

		expected := LLMResult{
			Type:      "tv",
			BaseTitle: "Test Show",
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponseBody(t, llmResultJSON(t, expected))))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 3,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	result, err := c.Classify(context.Background(), "Test Show S01E01", []TorrentFile{}, "testhash")
	require.NoError(t, err)
	require.Equal(t, 2, callCount)
	require.Equal(t, "tv", result.Type)
	require.Equal(t, "Test Show", result.BaseTitle)
}

func TestClassify_AllAttemptsFail(t *testing.T) {
	t.Parallel()

	var callCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++

		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"message": "rate limited"}}`))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 2,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test", []TorrentFile{}, "testhash")
	require.Error(t, err)
	require.Contains(t, err.Error(), "llm classify failed after")
	require.Equal(t, 3, callCount)
}

func TestClassify_AllAttemptsFail_LogsWarn(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"message": "rate limited"}}`))
	}))
	defer server.Close()

	core, logs := observer.New(zapcore.DebugLevel)

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 2,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.New(core).Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test", []TorrentFile{}, "testhash")
	require.Error(t, err)

	warns := logs.FilterLevelExact(zapcore.WarnLevel).All()
	require.Len(t, warns, 1)
	require.Equal(t, "llm classify failed", warns[0].Message)

	ctxMap := warns[0].ContextMap()
	require.Equal(t, testModel, ctxMap["model"])
	require.Equal(t, "testhash", ctxMap["info_hash"])
	require.Equal(t, int64(3), ctxMap["attempts"])
	require.Contains(t, fmt.Sprint(ctxMap["error"]), "rate limited")
}

func TestClassify_Success_NoWarn(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		expected := LLMResult{Type: testMovieType, BaseTitle: "Test Movie"}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponseBody(t, llmResultJSON(t, expected))))
	}))
	defer server.Close()

	core, logs := observer.New(zapcore.DebugLevel)

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 0,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.New(core).Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test", []TorrentFile{}, "testhash")
	require.NoError(t, err)
	require.Empty(t, logs.FilterLevelExact(zapcore.WarnLevel).All())
}

func TestNewClient_SamplesRepeatedWarns(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"message": "rate limited"}}`))
	}))
	defer server.Close()

	core, logs := observer.New(zapcore.DebugLevel)

	c := NewClient(LLMConfig{Model: testModel, MaxRetries: 0, Timeout: 10}, zap.New(core).Sugar())
	c.http = resty.New().SetBaseURL(server.URL)

	for range llmWarnLogBurst * 2 {
		_, err := c.Classify(context.Background(), "Test", []TorrentFile{}, "testhash")
		require.Error(t, err)
	}

	require.Equal(t, llmWarnLogBurst, logs.FilterLevelExact(zapcore.WarnLevel).Len())
}

func TestClassify_EmptyChoices(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices": []}`))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 0,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test", []TorrentFile{}, "testhash")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no choices")
}

func TestClassify_APIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error": {"message": "rate limit exceeded"}}`))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 0,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test", []TorrentFile{}, "testhash")
	require.Error(t, err)
	require.Contains(t, err.Error(), "api error")
}

func TestClassify_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 0,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test", []TorrentFile{}, "testhash")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode response")
}

func TestClassify_EmptyContent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponseBody(t, "")))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 0,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test", []TorrentFile{}, "testhash")
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty response content")
}

func TestClassify_UnparseableLLMResult(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponseBody(t, `{"type": 123, "base_title": 456}`)))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 0,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test", []TorrentFile{}, "testhash")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse llm output")
}

func TestClassify_UsesJSONSchemaResponseFormat(t *testing.T) {
	t.Parallel()

	var reqBodyBytes []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBodyBytes, _ = io.ReadAll(r.Body)

		expected := LLMResult{Type: testMovieType, BaseTitle: "Test"}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponseBody(t, llmResultJSON(t, expected))))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 0,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test Movie 2024", []TorrentFile{}, "testhash")
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(reqBodyBytes, &body))

	format, ok := body["response_format"].(map[string]any)
	require.True(t, ok, "response_format should be present")
	require.Equal(t, "json_schema", format["type"])

	jsonSchema, ok := format["json_schema"].(map[string]any)
	require.True(t, ok, "json_schema should be present")
	require.Equal(t, llmResultSchemaName, jsonSchema["name"])
	require.Equal(t, true, jsonSchema["strict"])

	schema, ok := jsonSchema["schema"].(map[string]any)
	require.True(t, ok, "schema should be present")
	require.Equal(t, "object", schema["type"])
	require.Equal(t, false, schema["additionalProperties"])
	require.ElementsMatch(t,
		[]any{"type", "base_title", "date", "languages"},
		schema["required"],
	)

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "properties should be present")

	typeProp, ok := properties["type"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "string", typeProp["type"])
	require.ElementsMatch(t, []any{
		"movie", "tv_show", "music", "ebook", "comic", "audiobook",
		"game", "software", "adult", "other", "unknown",
	}, typeProp["enum"])

	baseTitleProp, ok := properties["base_title"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "string", baseTitleProp["type"])

	dateProp, ok := properties["date"].(map[string]any)
	require.True(t, ok)
	require.ElementsMatch(t, []any{"string", "null"}, dateProp["type"])

	languagesProp, ok := properties["languages"].(map[string]any)
	require.True(t, ok)
	require.ElementsMatch(t, []any{"array", "null"}, languagesProp["type"])
}

func TestClassify_ReasoningEffortNone_DisablesThinking(t *testing.T) {
	t.Parallel()

	var (
		reqBodyBytes []byte
		gotHeader    string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBodyBytes, _ = io.ReadAll(r.Body)
		gotHeader = r.Header.Get("X-Bf-Passthrough-Extra-Params")

		expected := LLMResult{Type: testMovieType, BaseTitle: "Test"}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponseBody(t, llmResultJSON(t, expected))))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:           testModel,
			MaxRetries:      0,
			Timeout:         10,
			ReasoningEffort: reasoningEffortNone,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test Movie 2024", []TorrentFile{}, "testhash")
	require.NoError(t, err)
	require.Equal(t, "true", gotHeader)

	var body map[string]any
	require.NoError(t, json.Unmarshal(reqBodyBytes, &body))
	require.Equal(t, reasoningEffortNone, body["reasoning_effort"])
	require.InDelta(t, float64(0), body["reasoning_budget"], 1e-6)

	ctk, ok := body["chat_template_kwargs"].(map[string]any)
	require.True(t, ok, "chat_template_kwargs should be present")
	require.Equal(t, false, ctk["enable_thinking"])

	thinking, ok := body["thinking"].(map[string]any)
	require.True(t, ok, "thinking should be present")
	require.Equal(t, "disabled", thinking["type"])
}

func TestClassify_ReasoningEffortLow_NoExtraParams(t *testing.T) {
	t.Parallel()

	var (
		reqBodyBytes []byte
		gotHeader    string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBodyBytes, _ = io.ReadAll(r.Body)
		gotHeader = r.Header.Get("X-Bf-Passthrough-Extra-Params")

		expected := LLMResult{Type: testMovieType, BaseTitle: "Test"}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponseBody(t, llmResultJSON(t, expected))))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:           testModel,
			MaxRetries:      0,
			Timeout:         10,
			ReasoningEffort: "low",
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test Movie 2024", []TorrentFile{}, "testhash")
	require.NoError(t, err)
	require.Empty(t, gotHeader)

	var body map[string]any
	require.NoError(t, json.Unmarshal(reqBodyBytes, &body))
	require.Equal(t, "low", body["reasoning_effort"])
	require.NotContains(t, body, "reasoning_budget")
	require.NotContains(t, body, "chat_template_kwargs")
	require.NotContains(t, body, "thinking")
}

func TestClassify_UsesCustomPrompt(t *testing.T) {
	t.Parallel()

	var (
		reqBodyBytes []byte
		gotHeader    string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBodyBytes, _ = io.ReadAll(r.Body)
		gotHeader = r.Header.Get("X-Bf-Passthrough-Extra-Params")

		expected := LLMResult{Type: testMovieType, BaseTitle: "Test"}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponseBody(t, llmResultJSON(t, expected))))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:           testModel,
			MaxRetries:      0,
			Timeout:         10,
			ReasoningEffort: reasoningEffortNone,
			Prompt:          "Custom classifier instructions",
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	_, err := c.Classify(context.Background(), "Test Movie 2024", []TorrentFile{}, "testhash")
	require.NoError(t, err)
	require.Equal(t, "true", gotHeader)

	var body map[string]any
	require.NoError(t, json.Unmarshal(reqBodyBytes, &body))

	messages, ok := body["messages"].([]any)
	require.True(t, ok, "messages should be present")
	require.Len(t, messages, 2)

	systemMsg, ok := messages[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Custom classifier instructions", systemMsg["content"])
}

func TestClassify_CodeFencedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		expected := LLMResult{Type: testMovieType, BaseTitle: "Test"}
		content := "```json\n" + llmResultJSON(t, expected) + "\n```"

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(chatResponseBody(t, content)))
	}))
	defer server.Close()

	c := &Client{
		config: LLMConfig{
			Model:      testModel,
			MaxRetries: 0,
			Timeout:    10,
		},
		http:   resty.New().SetBaseURL(server.URL),
		logger: zap.NewNop().Sugar(),
	}

	result, err := c.Classify(context.Background(), "Test Movie 2024", []TorrentFile{}, "testhash")
	require.NoError(t, err)
	require.Equal(t, testMovieType, result.Type)
	require.Equal(t, "Test", result.BaseTitle)
}
