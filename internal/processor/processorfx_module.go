package processor

import (
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/hexsans/hexmagnet/internal/retryqueue"
	"go.uber.org/fx"
)

func NewProcessorFxModule() fx.Option {
	return fx.Module(
		"processor",
		retryqueue.NewModule(),
		fx.Provide(
			New,
			NewConsumer,
			indexer.New,
			indexer.NewDeleter,
		),
	)
}
