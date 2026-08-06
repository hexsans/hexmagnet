package db

import (
	"context"

	"github.com/hexsans/hexmagnet/internal/database/migrations"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	PoolRef  *postgres.PoolRef
	Migrator utils.Lazy[migrations.Migrator]
	Logger   *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Queries utils.Lazy[*Queries]
}

func NewFx(p Params) Result {
	return Result{
		Queries: utils.NewLazy(func() (*Queries, error) {
			m, err := p.Migrator.Get()
			if err != nil {
				return nil, err
			}

			if migrateErr := m.Up(context.Background()); migrateErr != nil {
				return nil, migrateErr
			}

			pool := p.PoolRef.Get()
			q := NewQueries(pool)

			return q, nil
		}),
	}
}
