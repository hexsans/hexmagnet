package esearch

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/zap"
)

const (
	indexName = "torrent_content"

	esKeyBool        = "bool"
	esKeyContentType = "content_type"
	esKeyCreatedAt   = "created_at"
	esKeyDesc        = "desc"
	esKeyGTE         = "gte"
	esKeyInfoHash    = "info_hash"
	esKeyScore       = "_score"
	esKeyTerms       = "terms"
	esKeyTerm        = "term"
	esKeySize        = "size"
)

type ESearch struct {
	client             *elasticsearch.Client
	q                  *db.Queries
	embedder           embedding.Embedder
	logger             *zap.SugaredLogger
	instructionEnabled bool
}

func New(
	client *elasticsearch.Client,
	q *db.Queries,
	embedder embedding.Embedder,
	logger *zap.SugaredLogger,
	instructionEnabled bool,
) *ESearch {
	return &ESearch{client: client, q: q, embedder: embedder, logger: logger, instructionEnabled: instructionEnabled}
}

func (s *ESearch) TorrentSearch(ctx context.Context, params search.TorrentSearchParams) (search.TorrentSearchResult, error) {
	if params.Barrier != "" {
		if decoded, err := search.DecodeBarrier(params.Barrier); err == nil {
			params.Barrier = decoded
		}
	}

	body := map[string]any{}

	// Build query
	boolQuery := map[string]any{}

	var (
		shouldClauses []map[string]any
		filterClauses []map[string]any
	)

	// Text search
	var knnClause map[string]any

	if params.QueryString != "" {
		shouldClauses = append(shouldClauses, map[string]any{
			esKeyBool: map[string]any{
				"must": []map[string]any{
					{
						"multi_match": map[string]any{
							"query":                params.QueryString,
							"fields":               []string{"name^3", "title^2", "overview", "search_text"},
							"type":                 "best_fields",
							"operator":             "or",
							"minimum_should_match": "50%",
						},
					},
				},
				"should": []map[string]any{
					{
						"multi_match": map[string]any{
							"query":    params.QueryString,
							"fields":   []string{"name^3", "title^2", "overview", "search_text"},
							"type":     "best_fields",
							"operator": "and",
							"boost":    3,
						},
					},
				},
			},
		})

		if s.embedder != nil {
			queryText := params.QueryString
			if s.instructionEnabled {
				queryText = "Instruct: Retrieve semantically relevant torrents in any language for the query.\nQuery: " + queryText
			}

			vector, err := s.embedder.EmbedSingle(ctx, queryText)
			if err == nil {
				k := int(params.Limit)
				if k == 0 {
					k = 10
				}

				rrfWindowK := k * 20
				if rrfWindowK < 200 {
					rrfWindowK = 200
				}

				numCandidates := rrfWindowK * 10
				if numCandidates < 200 {
					numCandidates = 200
				}

				knnClause = map[string]any{
					"knn": map[string]any{
						"field":          "search_vector",
						"query_vector":   vector,
						"k":              rrfWindowK,
						"num_candidates": numCandidates,
					},
				}
			}
		}
	}

	// Filters
	if len(params.InfoHashes) > 0 {
		filterClauses = append(filterClauses, map[string]any{
			esKeyTerms: map[string]any{esKeyInfoHash: params.InfoHashes},
		})
	}

	if len(params.Languages) > 0 {
		filterClauses = append(filterClauses, map[string]any{
			esKeyTerms: map[string]any{"languages": params.Languages},
		})
	}

	if len(params.ReleaseYears) > 0 {
		filterClauses = append(filterClauses, map[string]any{
			esKeyTerms: map[string]any{"release_year": toAnySlice(params.ReleaseYears)},
		})
	}

	if len(params.ContentRefs) > 0 {
		refClauses := make([]map[string]any, 0, len(params.ContentRefs))
		for _, ref := range params.ContentRefs {
			refClauses = append(refClauses, map[string]any{
				esKeyBool: map[string]any{
					"must": []map[string]any{
						{esKeyTerm: map[string]any{esKeyContentType: ref.Type}},
						{esKeyTerm: map[string]any{"content_source": ref.Source}},
						{esKeyTerm: map[string]any{"content_id": ref.ID}},
					},
				},
			})
		}

		filterClauses = append(filterClauses, map[string]any{
			esKeyBool: map[string]any{"should": refClauses, "minimum_should_match": 1},
		})
	}

	barrier := params.Barrier
	if barrier == "" {
		barrier = time.Now().UTC().Add(-2 * time.Second).Format(time.RFC3339)
	}

	filterClauses = append(filterClauses, map[string]any{
		"range": map[string]any{
			esKeyCreatedAt: map[string]any{
				"lte": barrier,
			},
		},
	})

	if len(shouldClauses) > 0 || len(filterClauses) > 0 {
		q := map[string]any{}
		if len(shouldClauses) > 0 {
			q["should"] = shouldClauses
			q["minimum_should_match"] = 1
		}

		if len(filterClauses) > 0 {
			q["filter"] = filterClauses
		}

		boolQuery[esKeyBool] = q
		body["query"] = boolQuery
	} else {
		for k, v := range knnClause {
			body[k] = v
		}
	}

	// Sort — deterministic ordering with info_hash as unique tiebreaker.
	if len(params.OrderBy) > 0 {
		var sortClauses []any

		for _, ob := range params.OrderBy {
			field := esSortField(ob.Field)
			if field == "" {
				continue
			}

			order := "asc"
			if ob.Direction == search.SortDesc {
				order = esKeyDesc
			}

			sortClauses = append(sortClauses, map[string]any{field: order})
		}

		if len(sortClauses) > 0 {
			sortClauses = append(
				sortClauses,
				map[string]any{esKeyCreatedAt: esKeyDesc},
				map[string]any{esKeyInfoHash: esKeyDesc},
			)
			body["sort"] = sortClauses
		}
	} else {
		if params.QueryString != "" {
			body["sort"] = []map[string]any{{esKeyScore: esKeyDesc}, {esKeyCreatedAt: esKeyDesc}, {esKeyInfoHash: esKeyDesc}}
		} else {
			body["sort"] = []map[string]any{{esKeyCreatedAt: esKeyDesc}, {esKeyScore: esKeyDesc}, {esKeyInfoHash: esKeyDesc}}
		}
	}

	// Pagination
	limit := int(params.Limit)
	if limit == 0 {
		limit = 10
	}

	offset := int(params.Offset)
	body["from"] = offset
	body[esKeySize] = limit

	body["track_total_hits"] = true

	// Aggregations
	if hasFacets(params.FacetAggregate) {
		body["aggs"] = buildAggs(params)
	}

	// Execute search
	if knnClause != nil && len(shouldClauses) > 0 {
		rrfWindow := limit * 20
		if rrfWindow < 200 {
			rrfWindow = 200
		}

		// Save original pagination to restore on fallback
		origFrom := body["from"]
		origSize := body[esKeySize]

		// BM25 search
		body["from"] = 0
		body[esKeySize] = rrfWindow

		bm25Result, bm25Err := s.client.Search(ctx, indexName, body)
		if bm25Err != nil {
			return search.TorrentSearchResult{}, fmt.Errorf("es bm25 search: %w", bm25Err)
		}

		// kNN search
		knn := knnClause["knn"].(map[string]any)

		knnCopy := make(map[string]any, len(knn)+1)
		for k, v := range knn {
			knnCopy[k] = v
		}

		if len(filterClauses) > 0 {
			knnCopy["filter"] = map[string]any{
				esKeyBool: map[string]any{"filter": filterClauses},
			}
		}

		knnBody := map[string]any{
			"knn":              knnCopy,
			esKeySize:          rrfWindow,
			"track_total_hits": true,
		}
		if sort, ok := body["sort"]; ok {
			knnBody["sort"] = sort
		}

		knnResult, knnErr := s.client.Search(ctx, indexName, knnBody)
		if knnErr != nil {
			if s.logger != nil {
				s.logger.Debugw("kNN search failed, degrading to BM25-only",
					"query", params.QueryString, "error", knnErr)
			}
		}

		// Filter low-scoring kNN hits
		if knnErr == nil && len(knnResult.Hits.Hits) > 0 {
			var topScore float64
			for _, hit := range knnResult.Hits.Hits {
				if hit.Score != nil && *hit.Score > topScore {
					topScore = *hit.Score
				}
			}

			threshold := topScore * 0.6
			if threshold < 0.3 {
				threshold = 0.3
			}

			filtered := knnResult.Hits.Hits[:0]
			for _, hit := range knnResult.Hits.Hits {
				if hit.Score != nil && *hit.Score >= threshold {
					filtered = append(filtered, hit)
				}
			}

			knnResult.Hits.Hits = filtered
			if len(knnResult.Hits.Hits) == 0 {
				knnErr = fmt.Errorf("kNN top score %f below threshold 0.3", topScore)
				if s.logger != nil {
					s.logger.Debugw("kNN score too low, degrading to BM25-only",
						"query", params.QueryString, "topScore", topScore)
				}
			}
		}

		// RRF merge
		if knnErr == nil {
			rrf := rrfMerge(bm25Result.Hits.Hits, knnResult.Hits.Hits, rrfWindow, limit, offset, params.ContentTypes)

			items := make([]search.TorrentSearchRow, 0, len(rrf.hits))
			for _, hit := range rrf.hits {
				row, err := hitToRow(hit.Source)
				if err != nil {
					return search.TorrentSearchResult{}, fmt.Errorf("map hit: %w", err)
				}

				items = append(items, row)
			}

			aggs := map[string]search.AggregationBucket{}

			if bm25Result.Aggs != nil {
				var err error

				aggs, err = parseAggs(bm25Result.Aggs, params)
				if err != nil {
					return search.TorrentSearchResult{}, fmt.Errorf("parse aggs: %w", err)
				}
			}

			aggs[esKeyContentType] = rrf.ctBuckets

			totalCount := uint(rrf.total)
			if uint64(totalCount) != uint64(rrf.total) {
				totalCount = math.MaxUint32
			}

			totalCountIsEstimate := false
			if bm25Result.Hits.Total.Relation == esKeyGTE || knnResult.Hits.Total.Relation == esKeyGTE {
				totalCountIsEstimate = true
			}

			hasNextPage := len(items) > 0 && offset+len(items) < int(totalCount)

			return search.TorrentSearchResult{
				Items:                items,
				TotalCount:           totalCount,
				TotalCountIsEstimate: totalCountIsEstimate,
				HasNextPage:          hasNextPage,
				Barrier:              search.EncodeBarrier(barrier),
				Aggregations:         aggs,
			}, nil
		}

		// Restore original pagination when falling through to BM25-only
		body["from"] = origFrom
		body[esKeySize] = origSize
	}

	// For non-RRF path, add content_type filter
	if len(params.ContentTypes) > 0 {
		ctFilter := map[string]any{
			esKeyTerms: map[string]any{esKeyContentType: params.ContentTypes},
		}
		if params.FacetAggregate.ContentType {
			// Use post_filter so aggregations see unfiltered counts (just like the PostgreSQL path)
			body["post_filter"] = ctFilter
		} else if q, ok := body["query"].(map[string]any); ok {
			if bq, ok := q[esKeyBool].(map[string]any); ok {
				if existing, ok := bq["filter"]; ok {
					bq["filter"] = append(existing.([]map[string]any), ctFilter)
				} else {
					bq["filter"] = []map[string]any{ctFilter}
				}
			}
		}
	}

	result, err := s.client.Search(ctx, indexName, body)
	if err != nil {
		return search.TorrentSearchResult{}, fmt.Errorf("es search: %w", err)
	}

	// Map hits
	items := make([]search.TorrentSearchRow, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		row, err := hitToRow(hit.Source)
		if err != nil {
			return search.TorrentSearchResult{}, fmt.Errorf("map hit: %w", err)
		}

		items = append(items, row)
	}

	// Aggregations
	aggs := map[string]search.AggregationBucket{}
	if result.Aggs != nil {
		aggs, err = parseAggs(result.Aggs, params)
		if err != nil {
			return search.TorrentSearchResult{}, fmt.Errorf("parse aggs: %w", err)
		}
	}

	// Total count
	totalCount := uint(0)
	if result.Hits.Total.Relation != "" && result.Hits.Total.Value > 0 {
		totalCount = uint(result.Hits.Total.Value)
		if uint64(totalCount) != uint64(result.Hits.Total.Value) {
			totalCount = math.MaxUint32
		}
	}

	totalCountIsEstimate := result.Hits.Total.Relation == esKeyGTE

	// If content_type filter is active via post_filter, compute filtered total from aggregation
	if len(params.ContentTypes) > 0 && params.FacetAggregate.ContentType {
		if ctBucket, ok := aggs[esKeyContentType]; ok {
			filteredCount := uint(0)

			for _, ct := range params.ContentTypes {
				if item, ok := ctBucket.Items[ct]; ok {
					filteredCount += item.Count
				}
			}

			if filteredCount > 0 {
				totalCount = filteredCount
				totalCountIsEstimate = false
			}
		}
	}

	// Has next page
	hasNextPage := len(items) > 0 && offset+len(items) < int(totalCount)

	return search.TorrentSearchResult{
		Items:                items,
		TotalCount:           totalCount,
		TotalCountIsEstimate: totalCountIsEstimate,
		HasNextPage:          hasNextPage,
		Barrier:              search.EncodeBarrier(barrier),
		Aggregations:         aggs,
	}, nil
}

func esSortField(f search.TorrentSearchField) string {
	switch f {
	case search.FieldCreatedAt:
		return esKeyCreatedAt
	case search.FieldUpdatedAt:
		return "updated_at"
	case search.FieldSize:
		return esKeySize
	case search.FieldFilesCount:
		return "files_count"
	case search.FieldSeeders:
		return "seeders"
	case search.FieldLeechers:
		return "leechers"
	case search.FieldName:
		return "name.keyword"
	case search.FieldInfoHash:
		return esKeyInfoHash
	case search.FieldRelevance:
		return esKeyScore
	}

	return ""
}

func hitToRow(src json.RawMessage) (search.TorrentSearchRow, error) {
	var doc struct {
		ID            string   `json:"id"`
		InfoHash      string   `json:"info_hash"`
		Name          string   `json:"name"`
		Title         string   `json:"title"`
		Overview      string   `json:"overview"`
		ContentType   string   `json:"content_type"`
		ContentSource string   `json:"content_source"`
		ContentID     string   `json:"content_id"`
		Size          int64    `json:"size"`
		Seeders       *uint    `json:"seeders"`
		Leechers      *uint    `json:"leechers"`
		FilesCount    *uint    `json:"files_count"`
		Languages     []string `json:"languages"`
		Private       bool     `json:"private"`
		CreatedAt     string   `json:"created_at"`
		UpdatedAt     string   `json:"updated_at"`
	}

	if err := json.Unmarshal(src, &doc); err != nil {
		return search.TorrentSearchRow{}, fmt.Errorf("unmarshal: %w", err)
	}

	var (
		contentTitle    *string
		contentOverview *string
	)

	if doc.Title != "" {
		contentTitle = &doc.Title
	}

	if doc.Overview != "" {
		contentOverview = &doc.Overview
	}

	var contentPtr *string
	if doc.ContentType != "" && doc.ContentType != "unknown" {
		contentPtr = &doc.ContentType
	}

	var contentSrcPtr *string
	if doc.ContentSource != "" {
		contentSrcPtr = &doc.ContentSource
	}

	var contentIDPtr *string
	if doc.ContentID != "" {
		contentIDPtr = &doc.ContentID
	}

	var seeders *int32

	if doc.Seeders != nil {
		v := utils.ClampInt32(*doc.Seeders)
		seeders = &v
	}

	var leechers *int32

	if doc.Leechers != nil {
		v := utils.ClampInt32(*doc.Leechers)
		leechers = &v
	}

	var filesCount *int32

	if doc.FilesCount != nil {
		v := utils.ClampInt32(*doc.FilesCount)
		filesCount = &v
	}

	createdAt := parseISO(doc.CreatedAt)
	updatedAt := parseISO(doc.UpdatedAt)

	return search.TorrentSearchRow{
		InfoHash:        doc.InfoHash,
		ContentType:     contentPtr,
		ContentSource:   contentSrcPtr,
		ContentID:       contentIDPtr,
		Languages:       stringSliceToBytes(doc.Languages),
		Seeders:         seeders,
		Leechers:        leechers,
		Size:            doc.Size,
		FilesCount:      filesCount,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		TorrentName:     doc.Name,
		TorrentPrivate:  doc.Private,
		ContentTitle:    contentTitle,
		ContentOverview: contentOverview,
	}, nil
}

func parseISO(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z", s)
		if err != nil {
			return time.Time{}
		}
	}

	return t
}

func stringSliceToBytes(ss []string) []byte {
	if len(ss) == 0 {
		return nil
	}
	// JSON marshal the slice as a compact JSON array
	b, err := json.Marshal(ss)
	if err != nil {
		return nil
	}

	return b
}

func toAnySlice[T any](s []T) []any {
	r := make([]any, len(s))
	for i, v := range s {
		r[i] = v
	}

	return r
}

type rrfMergeResult struct {
	hits      []elasticsearch.SearchHit
	total     int
	ctBuckets search.AggregationBucket
}

func rrfMerge(bm25Hits, knnHits []elasticsearch.SearchHit, windowSize, limit, offset int, filterCTs []string) rrfMergeResult {
	rrfScores := map[string]float64{}
	hitByID := map[string]elasticsearch.SearchHit{}

	for i, hit := range bm25Hits {
		if i >= windowSize {
			break
		}

		rrfScores[hit.ID] += 3.0 / float64(i+1+30)
		if _, ok := hitByID[hit.ID]; !ok {
			hitByID[hit.ID] = hit
		}
	}

	for i, hit := range knnHits {
		if i >= windowSize {
			break
		}

		rrfScores[hit.ID] += 3.0 / float64(i+1+60)
		if _, ok := hitByID[hit.ID]; !ok {
			hitByID[hit.ID] = hit
		}
	}

	// Count content_type from all unique hits (unfiltered)
	ctCount := map[string]uint{}

	for _, hit := range hitByID {
		var doc struct {
			ContentType string `json:"content_type"`
		}
		if err := json.Unmarshal(hit.Source, &doc); err == nil && doc.ContentType != "" {
			ctCount[doc.ContentType]++
		}
	}

	ctItems := search.AggregationItems{}
	for ct, count := range ctCount {
		ctItems[ct] = search.AggregationItem{Count: count, Label: ct}
	}

	type entry struct {
		hit         elasticsearch.SearchHit
		score       float64
		contentType string
	}

	entries := make([]entry, 0, len(rrfScores))
	for id, score := range rrfScores {
		e := entry{hit: hitByID[id], score: score}

		var doc struct {
			ContentType string `json:"content_type"`
		}
		if err := json.Unmarshal(e.hit.Source, &doc); err == nil {
			e.contentType = doc.ContentType
		}

		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})

	// Filter by content_type
	if len(filterCTs) > 0 {
		filtered := entries[:0]
		for _, e := range entries {
			for _, ct := range filterCTs {
				if e.contentType == ct {
					filtered = append(filtered, e)
					break
				}
			}
		}

		entries = filtered
	}

	totalItems := len(entries)
	if offset >= totalItems {
		return rrfMergeResult{
			hits:      nil,
			total:     totalItems,
			ctBuckets: search.AggregationBucket{Items: ctItems},
		}
	}

	end := offset + limit
	if end > totalItems {
		end = totalItems
	}

	result := make([]elasticsearch.SearchHit, 0, end-offset)
	for _, e := range entries[offset:end] {
		result = append(result, e.hit)
	}

	return rrfMergeResult{
		hits:      result,
		total:     totalItems,
		ctBuckets: search.AggregationBucket{Items: ctItems},
	}
}

func hasFacets(cfg search.FacetAggregationConfig) bool {
	return cfg.ContentType || cfg.FileType ||
		cfg.Language || cfg.ReleaseYear
}

func buildAggs(params search.TorrentSearchParams) map[string]any {
	aggs := map[string]any{}

	if params.FacetAggregate.ContentType {
		aggs[esKeyContentType] = buildSimpleAgg(esKeyContentType)
	}

	if params.FacetAggregate.FileType {
		aggs["file_type"] = buildSimpleAgg("file_types")
	}

	if params.FacetAggregate.Language {
		aggs["language"] = buildSimpleAgg("languages")
	}

	if params.FacetAggregate.ReleaseYear {
		aggs["release_year"] = buildSimpleAgg("release_year")
	}

	return aggs
}

func buildSimpleAgg(field string) map[string]any {
	return map[string]any{
		esKeyTerms: map[string]any{
			"field":   field,
			esKeySize: 100,
		},
	}
}

func parseAggs(raw map[string]json.RawMessage, params search.TorrentSearchParams) (map[string]search.AggregationBucket, error) {
	aggs := map[string]search.AggregationBucket{}

	// Check which facets are enabled
	enabled := map[string]bool{}
	if params.FacetAggregate.ContentType {
		enabled[esKeyContentType] = true
	}

	if params.FacetAggregate.FileType {
		enabled["file_type"] = true
	}

	if params.FacetAggregate.Language {
		enabled["language"] = true
	}

	if params.FacetAggregate.ReleaseYear {
		enabled["release_year"] = true
	}

	for name := range enabled {
		rawBucket, ok := raw[name]
		if !ok {
			continue
		}

		bucket := search.AggregationBucket{
			Items: search.AggregationItems{},
		}

		// Unwrap the aggregation response
		var buckets []aggBucketItem

		var simpleAgg struct {
			Buckets []aggBucketItem `json:"buckets"`
		}
		if err := json.Unmarshal(rawBucket, &simpleAgg); err != nil {
			continue
		}

		buckets = simpleAgg.Buckets

		for _, b := range buckets {
			bucket.Items[b.Key] = search.AggregationItem{
				Count: b.DocCount,
				Label: b.Key,
			}
		}

		aggs[name] = bucket
	}

	return aggs, nil
}

func (ESearch) Close() error { return nil }

// Ensure ESearch implements the query side consumed by search.Router.
var _ search.TorrentSearcher = (*ESearch)(nil)

type aggBucketItem struct {
	Key      string `json:"key"`
	DocCount uint   `json:"doc_count"`
}
