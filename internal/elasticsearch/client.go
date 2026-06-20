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

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Client struct {
	es *elasticsearch.Client
}

func NewClient(cfg Config) (*Client, error) {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     20,
		IdleConnTimeout:     90 * time.Second,
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
		Transport: transport,
	})
	if err != nil {
		return nil, fmt.Errorf("create elasticsearch client: %w", err)
	}

	return &Client{es: es}, nil
}

func (c *Client) doRequest(ctx context.Context, req esapi.Request, opName string) (*esapi.Response, error) {
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opName, err)
	}

	if res.IsError() {
		res.Body.Close()
		return nil, fmt.Errorf("%s: %s", opName, res.String())
	}

	return res, nil
}

func (c *Client) CreateIndex(ctx context.Context, name string, mapping io.Reader) error {
	req := esapi.IndicesCreateRequest{
		Index: name,
		Body:  mapping,
	}

	res, err := c.doRequest(ctx, req, "create index")
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

func (c *Client) IndexExists(ctx context.Context, name string) (bool, error) {
	req := esapi.IndicesExistsRequest{
		Index: []string{name},
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return false, fmt.Errorf("check index exists: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if res.IsError() {
		return false, fmt.Errorf("check index exists: %s", res.String())
	}

	return true, nil
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
	req := esapi.BulkRequest{
		Body: body,
	}

	res, err := c.doRequest(ctx, req, "bulk index")
	if err != nil {
		return err
	}
	defer res.Body.Close()

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
	req := esapi.IndicesDeleteRequest{
		Index: []string{name},
	}

	res, err := c.doRequest(ctx, req, "delete index")
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

func (c *Client) Delete(ctx context.Context, index string, id string) error {
	req := esapi.DeleteRequest{
		Index:      index,
		DocumentID: id,
	}

	res, err := c.doRequest(ctx, req, "delete document")
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	res, err := c.es.Ping(c.es.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ping: %s", res.String())
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

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(bytes.NewReader(bodyJSON)),
	)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search: %s", res.String())
	}

	var result SearchResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	return &result, nil
}
