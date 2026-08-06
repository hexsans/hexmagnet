package dbsearch

import (
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/search"
)

func NewFromPool(q *db.Queries) search.Search {
	return &pgSearch{q: q}
}

type pgSearch struct {
	q *db.Queries
}

func (*pgSearch) Close() error { return nil }
