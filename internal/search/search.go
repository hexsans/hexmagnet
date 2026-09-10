package search

import (
	"context"
	"time"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/model"
)

type Search interface {
	TorrentSearch(ctx context.Context, params TorrentSearchParams) (TorrentSearchResult, error)
	TorrentsWithMissingInfoHashes(
		ctx context.Context,
		params TorrentsWithMissingInfoHashesParams,
	) (TorrentsWithMissingInfoHashesResult, error)
	TorrentFiles(ctx context.Context, params TorrentFilesSearchParams) (TorrentFilesResult, error)
	Close() error
}

type Result[T any] struct {
	Items       []T
	TotalCount  uint
	HasNextPage bool
}

type TorrentSearchParams struct {
	QueryString    string
	Limit          uint
	Offset         uint
	TotalCount     bool
	HasNextPage    bool
	Barrier        string
	InfoHashes     []string
	ContentTypes   []string
	Cached         bool
	FileTypes      []string
	Languages      []string
	ReleaseYears   []int32
	ContentRefs    []ContentRef
	OrderBy        []TorrentSearchOrder
	FacetAggregate FacetAggregationConfig
}

type TorrentsWithMissingInfoHashesParams struct {
	InfoHashes []string
}

type TorrentsWithMissingInfoHashesResult struct {
	Torrents          []model.Torrent
	MissingInfoHashes []string
}

type TorrentSearchField string

const (
	FieldCreatedAt  TorrentSearchField = "created_at"
	FieldUpdatedAt  TorrentSearchField = "updated_at"
	FieldSize       TorrentSearchField = "size"
	FieldFilesCount TorrentSearchField = "files_count"
	FieldSeeders    TorrentSearchField = "seeders"
	FieldLeechers   TorrentSearchField = "leechers"
	FieldName       TorrentSearchField = "name"
	FieldInfoHash   TorrentSearchField = "info_hash"
	FieldRelevance  TorrentSearchField = "relevance"
)

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type TorrentSearchOrder struct {
	Field     TorrentSearchField
	Direction SortDirection
}

type FacetAggregationConfig struct {
	ContentType bool
	FileType    bool
	Language    bool
	ReleaseYear bool
}

type AggregationItem struct {
	Count      uint
	Label      string
	IsEstimate bool
}

type AggregationItems map[string]AggregationItem

type AggregationBucket struct {
	Items AggregationItems
}

type TorrentSearchResult struct {
	Items                []TorrentSearchRow
	TotalCount           uint
	TotalCountIsEstimate bool
	HasNextPage          bool
	Barrier              string
	Aggregations         map[string]AggregationBucket
}

type TorrentSearchRow struct {
	InfoHash         string
	ContentType      *string
	ContentSource    *string
	ContentID        *string
	Languages        []byte
	Tsv              string
	Seeders          *int32
	Leechers         *int32
	Size             int64
	FilesCount       *int32
	CreatedAt        time.Time
	UpdatedAt        time.Time
	TorrentName      string
	TorrentPrivate   bool
	ContentTitle     *string
	ContentOverview  *string
	ContentCreatedAt *time.Time
	ContentUpdatedAt *time.Time
}

type ContentRef struct {
	Type   string
	Source string
	ID     string
}

type TorrentFilesSearchParams struct {
	InfoHash    string
	Limit       uint
	Offset      uint
	TotalCount  bool
	HasNextPage bool
	OrderBy     string
}

type TorrentFilesResult = Result[db.TorrentFile]
