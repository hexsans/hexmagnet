package gqlmodel

import (
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/metrics/torrentmetrics"
	dbsearch "github.com/hexsans/hexmagnet/internal/search"
)

type TorrentQuery struct {
	DB                   *db.Queries
	Search               dbsearch.Search
	TorrentMetricsClient torrentmetrics.Client
}

type TorrentMutation struct {
	DB       *db.Queries
	ESClient *elasticsearch.Client
	Embedder *embedding.Client
}
