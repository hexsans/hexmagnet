package dhtcrawler

import (
	"context"
	"sync"

	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type storeParams struct {
	fx.In

	PgxPool utils.Lazy[*pgxpool.Pool]
	Runtime *Runtime `name:"dht_crawler_runtime"`
	Logger  *zap.SugaredLogger
}

type invokeParams struct {
	fx.In

	Lifecycle   fx.Lifecycle
	Store       *Store
	PgxPoolWait *sync.WaitGroup `name:"pgx_pool_wait"`
}

func NewRuntimeStoreFxModule() fx.Option {
	return fx.Module(
		"runtimestore",
		fx.Provide(
			func(p storeParams) *Store {
				return NewStore(p.PgxPool, p.Runtime, p.Logger)
			},
		),
		fx.Invoke(func(p invokeParams) {
			p.Lifecycle.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					if err := p.Store.Load(ctx); err != nil {
						return err
					}

					p.PgxPoolWait.Add(1)
					p.Store.StartSaveLoop(ctx)

					return nil
				},
				OnStop: func(context.Context) error {
					p.Store.Stop()
					p.PgxPoolWait.Done()

					return nil
				},
			})
		}),
	)
}
