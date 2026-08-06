package db

import (
	"context"
	"sync"

	sqlc "github.com/hexsans/hexmagnet/internal/database/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Queries struct {
	*sqlc.Queries
	pool *pgxpool.Pool
	dbtx sqlc.DBTX
	mu   sync.RWMutex
}

func NewQueries(pool *pgxpool.Pool) *Queries {
	return &Queries{Queries: sqlc.New(pool), pool: pool, dbtx: pool}
}

func NewQueriesWithDBTX(dbtx sqlc.DBTX) *Queries {
	return &Queries{Queries: sqlc.New(dbtx), dbtx: dbtx}
}

func (q *Queries) ReplacePool(pool *pgxpool.Pool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.pool = pool
	q.dbtx = pool
	q.Queries = sqlc.New(pool)
}

func (q *Queries) BeginTx(ctx context.Context) (pgx.Tx, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.pool.Begin(ctx)
}

func (q *Queries) Read(_ context.Context) sqlc.DBTX {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.dbtx
}

func (q *Queries) Pool() *pgxpool.Pool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.pool
}
