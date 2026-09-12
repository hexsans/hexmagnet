package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type Client struct {
	config LLMConfig
	http   *resty.Client
	logger *zap.SugaredLogger
}

func NewClient(cfg LLMConfig, logger *zap.SugaredLogger) *Client {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}

	return &Client{
		config: cfg,
		logger: logger,
		http: resty.New().
			SetBaseURL(endpoint).
			SetTimeout(time.Duration(cfg.Timeout)*time.Second).
			SetAuthToken(cfg.APIKey).
			SetHeader("Content-Type", "application/json"),
	}
}

func (c *Client) Classify(ctx context.Context, name string, files []TorrentFile, infoHash string) (*LLMResult, error) {
	systemMsg, userMsg := BuildPrompt(name, c.config.Prompt, files, c.config.MaxFiles)

	var lastErr error

	for i := range c.config.MaxRetries + 1 {
		result, err := c.tryClassify(ctx, systemMsg, userMsg)
		if err == nil {
			if i > 0 {
				c.logger.Debugw("llm classify succeeded on retry",
					"attempt", i+1,
					"max_attempts", c.config.MaxRetries+1,
					"info_hash", infoHash,
				)
			}

			return result, nil
		}

		lastErr = err
		c.logger.Debugw("llm classify attempt failed",
			"attempt", i+1,
			"max_attempts", c.config.MaxRetries+1,
			"info_hash", infoHash,
			"error", err,
		)
	}

	return nil, fmt.Errorf("llm classify failed after %d retries: %w", c.config.MaxRetries+1, lastErr)
}

const maxErrorBodyLen = 512

func truncateBody(body []byte) string {
	if len(body) <= maxErrorBodyLen {
		return string(body)
	}

	return string(body[:maxErrorBodyLen]) + "...(truncated)"
}

func (c *Client) tryClassify(ctx context.Context, systemMsg, userMsg string) (*LLMResult, error) {
	req := chatRequest{
		Model:       c.config.Model,
		Temperature: c.config.Temperature,
		Messages: []chatMessage{
			{Role: "system", Content: systemMsg},
			{Role: "user", Content: userMsg},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
	}

	if c.config.ReasoningEffort != "" {
		effort := c.config.ReasoningEffort
		req.ReasoningEffort = &effort

		if effort == reasoningEffortNone {
			budget := 0
			req.ReasoningBudget = &budget
			req.ChatTemplateKwargs = &chatTemplateKwargs{EnableThinking: false}
			req.Thinking = &thinking{Type: "disabled"}
		}
	}

	r := c.http.R().
		SetContext(ctx).
		SetBody(&req)

	if c.config.ReasoningEffort == reasoningEffortNone {
		r.SetHeader("x-bf-passthrough-extra-params", "true")
	}

	resp, err := r.Post("/chat/completions")
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("api returned status %d: %s", resp.StatusCode(), truncateBody(resp.Body()))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("api error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := stripCodeFence(chatResp.Choices[0].Message.Content)
	if content == "" {
		return nil, fmt.Errorf("empty response content")
	}

	var result LLMResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse llm output: %w", err)
	}

	return &result, nil
}

// stripCodeFence trims surrounding whitespace and unwraps a markdown code
// fence around a JSON payload, since some models ignore the compact-JSON
// instruction and still emit ```json blocks.
func stripCodeFence(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "```") {
		return content
	}

	content = strings.TrimSpace(strings.TrimPrefix(content, "```"))
	if len(content) >= 4 && strings.EqualFold(content[:4], "json") {
		content = content[4:]
	}

	content = strings.TrimSpace(content)
	content = strings.TrimSuffix(content, "```")

	return strings.TrimSpace(content)
}
