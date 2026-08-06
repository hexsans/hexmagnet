package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	EmbedSingle(ctx context.Context, text string) ([]float32, error)
}

type Client struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
	model      string
	dimensions int
}

func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		endpoint:   strings.TrimRight(cfg.Endpoint, "/"),
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		dimensions: cfg.Dimensions,
	}
}

type embedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type embedDataItem struct {
	Embedding []float32 `json:"embedding"`
}

type embedResponse struct {
	Data []embedDataItem `json:"data"`
}

func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(embedRequest{
		Model:      c.model,
		Input:      texts,
		Dimensions: c.dimensions,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed request: status %d", resp.StatusCode)
	}

	var er embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}

	vectors := make([][]float32, len(er.Data))
	for i, d := range er.Data {
		if c.dimensions > 0 && len(d.Embedding) != c.dimensions {
			return nil, fmt.Errorf(
				"embedding dimension mismatch: got %d, expected %d (config dimensions)",
				len(d.Embedding),
				c.dimensions,
			)
		}

		vectors[i] = d.Embedding
	}

	return vectors, nil
}

func (c *Client) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vectors, err := c.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(vectors) == 0 {
		return nil, nil
	}

	return vectors[0], nil
}
