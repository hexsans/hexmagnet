package gqlmodel

import (
	"context"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/protocol"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/zap"
)

type TorrentSearchQuery struct {
	Svc    dbsearch.Search
	Logger *zap.SugaredLogger
}

type TorrentContent struct {
	InfoHash      protocol.ID
	ContentType   model.NullContentType
	ContentSource model.NullString
	ContentID     model.NullString
	Title         string
	Languages     []model.Language `json:"omitempty"`
	SearchString  string
	Seeders       model.NullUint
	Leechers      model.NullUint
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Torrent       model.Torrent
	Content       *model.Content
}

func torrentSearchRowToGQL(row dbsearch.TorrentSearchRow) TorrentContent {
	t := model.Torrent{
		InfoHash:   db.ToProtocolID(row.InfoHash),
		Name:       row.TorrentName,
		Size:       uint64(row.Size),
		Private:    row.TorrentPrivate,
		FilesCount: fromInt32PtrNull(row.FilesCount),
		Seeders:    fromInt32PtrNull(row.Seeders),
		Leechers:   fromInt32PtrNull(row.Leechers),
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}

	title := row.TorrentName
	if row.ContentTitle != nil {
		title = *row.ContentTitle
	}

	c := TorrentContent{
		InfoHash:  db.ToProtocolID(row.InfoHash),
		Title:     title,
		Seeders:   fromInt32PtrNull(row.Seeders),
		Leechers:  fromInt32PtrNull(row.Leechers),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Torrent:   t,
	}

	if row.ContentType != nil {
		if ct, err := model.ParseContentType(*row.ContentType); err == nil {
			c.ContentType = model.NewNullContentType(ct)
		}
	}

	if row.ContentSource != nil {
		c.ContentSource = model.NewNullString(*row.ContentSource)
	}

	if row.ContentID != nil {
		c.ContentID = model.NewNullString(*row.ContentID)
	}

	if len(row.Languages) > 0 {
		var langs model.Languages
		if err := langs.UnmarshalJSON(row.Languages); err == nil {
			c.Languages = langs.Slice()
		}
	}

	if row.ContentTitle != nil {
		c.Content = &model.Content{
			Title:     *row.ContentTitle,
			Overview:  nullStrPtr(row.ContentOverview),
			CreatedAt: timePtrOrZero(row.ContentCreatedAt),
		}
	}

	return c
}

func fromInt32PtrNull(v *int32) model.NullUint {
	if v == nil {
		return model.NullUint{}
	}

	return model.NewNullUint(uint(*v))
}

func nullStrPtr(v *string) model.NullString {
	if v == nil {
		return model.NullString{}
	}

	return model.NewNullString(*v)
}

type TorrentSearchQueryInput struct {
	QueryString       model.NullString
	Limit             model.NullUint
	Page              model.NullUint
	Offset            model.NullUint
	TotalCount        model.NullBool
	HasNextPage       model.NullBool
	Cached            model.NullBool
	AggregationBudget model.NullFloat64
	Barrier           model.NullString
	InfoHashes        graphql.Omittable[[]protocol.ID]
	Facets            *gen.TorrentSearchFacetsInput
	OrderBy           []gen.TorrentSearchOrderByInput
}

type TorrentSearchResult struct {
	TotalCount           uint
	TotalCountIsEstimate bool
	HasNextPage          bool
	Barrier              string
	Items                []TorrentContent
	Aggregations         gen.TorrentSearchAggregations
}

func (t TorrentSearchQuery) Search(
	ctx context.Context,
	input TorrentSearchQueryInput,
) (TorrentSearchResult, error) {
	p := ExtractPagination(input.Limit, input.Page, input.Offset, input.TotalCount, input.HasNextPage)

	params := dbsearch.TorrentSearchParams{
		QueryString: input.QueryString.String,
		Limit:       p.Limit,
		Offset:      p.Offset,
		TotalCount:  p.TotalCount,
		HasNextPage: p.HasNextPage,
		Barrier:     input.Barrier.String,
	}

	if hashes, ok := input.InfoHashes.ValueOK(); ok {
		params.InfoHashes = make([]string, len(hashes))
		for i, h := range hashes {
			params.InfoHashes[i] = h.String()
		}
	}

	if input.Facets != nil {
		extractFacetParams(input.Facets, &params)
	}

	fullOrderBy := utils.NewInsertMap[dbsearch.TorrentSearchField, dbsearch.SortDirection]()

	for _, ob := range input.OrderBy {
		if ob.Field == gen.TorrentSearchOrderByFieldRelevance && !input.QueryString.Valid {
			continue
		}

		field, err := parseTorrentSearchOrderBy(ob.Field.String())
		if err != nil {
			return TorrentSearchResult{}, err
		}

		fullOrderBy.Set(field, dbsearch.SortDirection(ob.Direction))
	}

	for _, entry := range fullOrderBy.Entries() {
		params.OrderBy = append(params.OrderBy, dbsearch.TorrentSearchOrder{
			Field:     entry.Key,
			Direction: entry.Value,
		})
	}

	result, resultErr := t.Svc.TorrentSearch(ctx, params)
	if resultErr != nil {
		if t.Logger != nil {
			t.Logger.Debugw("torrent content search failed", "query", input.QueryString.String, "error", resultErr)
		}

		return TorrentSearchResult{}, resultErr
	}

	return transformTorrentSearchResult(result)
}

func extractFacetParams(input *gen.TorrentSearchFacetsInput, params *dbsearch.TorrentSearchParams) {
	if ct, ok := input.ContentType.ValueOK(); ok {
		if agg, ok := ct.Aggregate.ValueOK(); ok && *agg {
			params.FacetAggregate.ContentType = true
		}

		if filter, ok := ct.Filter.ValueOK(); ok {
			params.ContentTypes = make([]string, len(filter))
			for i, v := range filter {
				params.ContentTypes[i] = v.String()
			}
		}
	}

	if ft, ok := input.TorrentFileType.ValueOK(); ok {
		if agg, ok := ft.Aggregate.ValueOK(); ok && *agg {
			params.FacetAggregate.FileType = true
		}

		if filter, ok := ft.Filter.ValueOK(); ok {
			params.FileTypes = make([]string, len(filter))
			for i, v := range filter {
				params.FileTypes[i] = v.String()
			}
		}
	}

	if lang, ok := input.Language.ValueOK(); ok {
		if agg, ok := lang.Aggregate.ValueOK(); ok && *agg {
			params.FacetAggregate.Language = true
		}

		if filter, ok := lang.Filter.ValueOK(); ok {
			params.Languages = make([]string, len(filter))
			for i, v := range filter {
				params.Languages[i] = v.String()
			}
		}
	}

	if yr, ok := input.ReleaseYear.ValueOK(); ok {
		if agg, ok := yr.Aggregate.ValueOK(); ok && *agg {
			params.FacetAggregate.ReleaseYear = true
		}

		if filter, ok := yr.Filter.ValueOK(); ok {
			params.ReleaseYears = make([]int32, len(filter))
			for i, v := range filter {
				params.ReleaseYears[i] = int32(*v)
			}
		}
	}
}

func parseTorrentSearchOrderBy(s string) (dbsearch.TorrentSearchField, error) {
	switch s {
	case "relevance":
		return dbsearch.FieldRelevance, nil
	case "created_at":
		return dbsearch.FieldCreatedAt, nil
	case "updated_at":
		return dbsearch.FieldUpdatedAt, nil
	case "size":
		return dbsearch.FieldSize, nil
	case "files_count":
		return dbsearch.FieldFilesCount, nil
	case "seeders":
		return dbsearch.FieldSeeders, nil
	case "leechers":
		return dbsearch.FieldLeechers, nil
	case "name":
		return dbsearch.FieldName, nil
	case "info_hash":
		return dbsearch.FieldInfoHash, nil
	default:
		return "", fmt.Errorf("unknown torrent content order by field: %s", s)
	}
}

func transformTorrentSearchResult(
	result dbsearch.TorrentSearchResult,
) (TorrentSearchResult, error) {
	aggs, aggsErr := transformTorrentSearchAggregations(result.Aggregations)
	if aggsErr != nil {
		return TorrentSearchResult{}, aggsErr
	}

	items := make([]TorrentContent, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, torrentSearchRowToGQL(item))
	}

	return TorrentSearchResult{
		TotalCount:           result.TotalCount,
		TotalCountIsEstimate: result.TotalCountIsEstimate,
		HasNextPage:          result.HasNextPage,
		Barrier:              result.Barrier,
		Items:                items,
		Aggregations:         aggs,
	}, nil
}

func transformTorrentSearchAggregations(aggs map[string]dbsearch.AggregationBucket) (gen.TorrentSearchAggregations, error) {
	var (
		result gen.TorrentSearchAggregations
		err    error
	)

	result.ContentType, err = contentTypeAggs(aggs["content_type"].Items)
	if err != nil {
		return result, err
	}

	result.TorrentFileType, err = torrentFileTypeAggs(aggs["file_type"].Items)
	if err != nil {
		return result, err
	}

	result.Language, err = languageAggs(aggs["language"].Items)
	if err != nil {
		return result, err
	}

	result.ReleaseYear, err = releaseYearAggs(aggs["release_year"].Items)
	if err != nil {
		return result, err
	}

	return result, nil
}

func timePtrOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}

	return *t
}
