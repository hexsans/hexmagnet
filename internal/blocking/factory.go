package blocking

import (
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	Pool   utils.Lazy[*pgxpool.Pool]
	Logger *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Manager utils.Lazy[Manager]
}

func New(params Params) Result {
	lazyManager := utils.NewLazy[Manager](func() (Manager, error) {
		pool, err := params.Pool.Get()
		if err != nil {
			return nil, err
		}

		return &manager{
			q:      db.NewQueries(pool),
			logger: params.Logger.Named("blocking"),
		}, nil
	})

	return Result{
		Manager: lazyManager,
	}
}
