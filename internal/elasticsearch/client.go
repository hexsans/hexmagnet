package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v9"
)

type Client struct {
	es *elasticsearch.TypedClient
}

func NewClient(cfg Config) (*Client, error) {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     20,
		IdleConnTimeout:     90 * time.Second,
	}

	es, err := elasticsearch.NewTyped(
		elasticsearch.WithAddresses(cfg.Addresses...),
		elasticsearch.WithTransportOptions(elastictransport.WithTransport(transport)),
	)
	if err != nil {
		return nil, fmt.Errorf("create elasticsearch client: %w", err)
	}

	return &Client{es: es}, nil
}

func (c *Client) CreateIndex(ctx context.Context, name string, mapping io.Reader) error {
	_, err := c.es.Indices.Create(name).Raw(mapping).Do(ctx)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	return nil
}

func (c *Client) IndexExists(ctx context.Context, name string) (bool, error) {
	ok, err := c.es.Indices.Exists(name).Do(ctx)
	if err != nil {
		return false, fmt.Errorf("check index exists: %w", err)
	}

	return ok, nil
}

type bulkItemError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type bulkItemIndex struct {
	ID     string         `json:"_id"`
	Status int            `json:"status"`
	Error  *bulkItemError `json:"error"`
}

type bulkItem struct {
	Index bulkItemIndex `json:"index"`
}

type bulkResponse struct {
	Errors bool       `json:"errors"`
	Items  []bulkItem `json:"items"`
}

func (c *Client) BulkIndex(ctx context.Context, body io.Reader) error {
	res, err := c.es.Bulk().Raw(body).Perform(ctx)
	if err != nil {
		return fmt.Errorf("bulk index: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("bulk index: %s", res.Status)
		}

		return fmt.Errorf("bulk index: %s", strings.TrimSpace(string(body)))
	}

	var br bulkResponse
	if err := json.NewDecoder(res.Body).Decode(&br); err != nil {
		return fmt.Errorf("bulk index: decode response: %w", err)
	}

	if br.Errors {
		var errs []string

		for _, item := range br.Items {
			if item.Index.Error != nil {
				errs = append(
					errs,
					fmt.Sprintf("doc %s: [%s] %s", item.Index.ID, item.Index.Error.Type, item.Index.Error.Reason),
				)
			}
		}

		return fmt.Errorf("bulk index: %d errors: %s", len(errs), strings.Join(errs, "; "))
	}

	return nil
}

func (c *Client) DeleteIndex(ctx context.Context, name string) error {
	_, err := c.es.Indices.Delete(name).Do(ctx)
	if err != nil {
		return fmt.Errorf("delete index: %w", err)
	}

	return nil
}

func (c *Client) Delete(ctx context.Context, index string, id string) error {
	_, err := c.es.Delete(index, id).Do(ctx)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}

	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	ok, err := c.es.Ping().Do(ctx)
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	if !ok {
		return fmt.Errorf("ping: unexpected status code")
	}

	return nil
}

type SearchResult struct {
	Took     int                        `json:"took"`
	TimedOut bool                       `json:"timed_out"`
	Hits     SearchHits                 `json:"hits"`
	Aggs     map[string]json.RawMessage `json:"aggregations"`
}

type SearchTotal struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

type SearchHits struct {
	Total    SearchTotal `json:"total"`
	MaxScore *float64    `json:"max_score"`
	Hits     []SearchHit `json:"hits"`
}

type SearchHit struct {
	Index  string          `json:"_index"`
	ID     string          `json:"_id"`
	Score  *float64        `json:"_score"`
	Source json.RawMessage `json:"_source"`
}

func (c *Client) Search(ctx context.Context, index string, body map[string]any) (*SearchResult, error) {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal search body: %w", err)
	}

	res, err := c.es.Search().Index(index).Raw(bytes.NewReader(bodyJSON)).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	result := &SearchResult{
		Took:     int(res.Took),
		TimedOut: res.TimedOut,
		Aggs:     make(map[string]json.RawMessage, len(res.Aggregations)),
	}

	if res.Hits.Total != nil {
		result.Hits.Total.Value = res.Hits.Total.Value
		result.Hits.Total.Relation = res.Hits.Total.Relation.String()
	}

	result.Hits.MaxScore = (*float64)(res.Hits.MaxScore)

	for _, h := range res.Hits.Hits {
		hit := SearchHit{
			Index:  h.Index_,
			Source: h.Source_,
			Score:  (*float64)(h.Score_),
		}

		if h.Id_ != nil {
			hit.ID = *h.Id_
		}

		result.Hits.Hits = append(result.Hits.Hits, hit)
	}

	for name, agg := range res.Aggregations {
		b, err := json.Marshal(agg)
		if err != nil {
			return nil, fmt.Errorf("marshal aggregation %s: %w", name, err)
		}

		result.Aggs[name] = b
	}

	return result, nil
}
