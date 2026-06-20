package processor

import (
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"go.uber.org/fx"
)

func NewProcessorFxModule() fx.Option {
	return fx.Module(
		"processor",
		fx.Provide(
			New,
			NewConsumer,
			indexer.New,
			indexer.NewDeleter,
		),
	)
}
