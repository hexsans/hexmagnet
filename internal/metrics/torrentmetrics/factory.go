package torrentmetrics

import (
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Queries utils.Lazy[*db.Queries]
}

type Result struct {
	fx.Out
	Client utils.Lazy[Client]
}

func New(p Params) Result {
	return Result{
		Client: utils.NewLazy[Client](func() (Client, error) {
			q, err := p.Queries.Get()
			if err != nil {
				return nil, err
			}

			return client{q}, nil
		}),
	}
}
