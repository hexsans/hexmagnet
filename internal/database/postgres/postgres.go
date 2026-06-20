package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	Config Config
	Logger *zap.SugaredLogger
}

type Result struct {
	fx.Out
	PgxPool     utils.Lazy[*pgxpool.Pool]
	SQLDB       utils.Lazy[*sql.DB]
	PgxPoolWait *sync.WaitGroup `name:"pgx_pool_wait"`
	AppHook     fx.Hook         `group:"app_hooks"`
	PoolRef     *PoolRef
	Runtime     *Runtime
}

func New(p Params) (Result, error) {
	waitGroup := &sync.WaitGroup{}
	runtime := &Runtime{
		poolRef: NewPoolRef(nil),
		cfg:     p.Config,
	}

	lazyPool := utils.NewLazy(func() (*pgxpool.Pool, error) {
		pool, err := newPool(context.Background(), p.Config, p.Logger)
		if err != nil {
			return nil, err
		}

		runtime.poolRef.Replace(pool)

		return pool, nil
	})

	return Result{
		PgxPool: lazyPool,
		SQLDB: utils.NewLazy(func() (*sql.DB, error) {
			pool, err := lazyPool.Get()
			if err != nil {
				return nil, err
			}

			return stdlib.OpenDBFromPool(pool), nil
		}),
		PgxPoolWait: waitGroup,
		AppHook: fx.Hook{
			OnStop: func(context.Context) error {
				waitGroup.Wait()

				if pool := runtime.poolRef.Get(); pool != nil {
					pool.Close()
				}

				return nil
			},
		},
		PoolRef: runtime.PoolRef(),
		Runtime: runtime,
	}, nil
}

func waitForPing(ctx context.Context, logger *zap.SugaredLogger, pool *pgxpool.Pool) error {
	i := 0

	var err error

	for {
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}

		err = pool.Ping(ctx)
		if err == nil {
			logger.Infow("connected to database")
			return nil
		}

		i++
		if i > 10 {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			logger.Warnw("failed to ping database, retrying...", "error", err)
		}
	}

	return fmt.Errorf("timed out waiting for ping: %w", err)
}
