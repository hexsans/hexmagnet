package gqlfx

import (
	"github.com/hexsans/hexmagnet/internal/gql"
	"github.com/hexsans/hexmagnet/internal/gql/resolvers"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
)

type GqlConfigParams struct {
	fx.In
	ResolverRoot utils.Lazy[*resolvers.Resolver]
}

func NewGqlConfig(p GqlConfigParams) utils.Lazy[gql.Config] {
	return utils.NewLazy(func() (gql.Config, error) {
		root, err := p.ResolverRoot.Get()
		if err != nil {
			return gql.Config{}, err
		}

		return gql.Config{Resolvers: root}, nil
	})
}
