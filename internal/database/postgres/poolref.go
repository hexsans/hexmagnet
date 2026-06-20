package postgres

import (
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolRef struct {
	pool atomic.Pointer[pgxpool.Pool]
}

func NewPoolRef(pool *pgxpool.Pool) *PoolRef {
	r := &PoolRef{}
	r.pool.Store(pool)

	return r
}

func (r *PoolRef) Get() *pgxpool.Pool {
	return r.pool.Load()
}

func (r *PoolRef) Replace(pool *pgxpool.Pool) *pgxpool.Pool {
	return r.pool.Swap(pool)
}
