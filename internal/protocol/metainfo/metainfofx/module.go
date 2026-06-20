package metainfofx

import (
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/banning"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"go.uber.org/fx"
)

func New() fx.Option {
	return fx.Module(
		"metainfo",
		fx.Provide(
			metainforequester.New,
			banning.New,
		),
	)
}
