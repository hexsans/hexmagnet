package search

import (
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
)

type Factory func(cfg indexer.SearchConfig) Search

type Runtime struct {
	Backend      *concurrency.AtomicValue[string]
	Search       *concurrency.AtomicValue[Search]
	SearchConfig *concurrency.AtomicValue[indexer.SearchConfig]

	factory Factory
}

func NewRuntime(cfg indexer.SearchConfig) *Runtime {
	r := &Runtime{
		Backend:      &concurrency.AtomicValue[string]{},
		Search:       &concurrency.AtomicValue[Search]{},
		SearchConfig: &concurrency.AtomicValue[indexer.SearchConfig]{},
	}
	r.Backend.Set(cfg.Backend)
	r.SearchConfig.Set(cfg)

	return r
}

func (r *Runtime) SetSearchFactory(fn Factory) {
	r.factory = fn
}

func (r *Runtime) SwitchBackend(cfg indexer.SearchConfig) {
	r.Backend.Set(cfg.Backend)
	r.SearchConfig.Set(cfg)

	if r.factory != nil {
		if old := r.Search.Get(); old != nil {
			_ = old.Close()
		}

		if s := r.factory(cfg); s != nil {
			r.Search.Set(s)
		}
	}
}
