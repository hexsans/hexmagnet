package classifier

import (
	"fmt"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Dependencies struct {
	Search     LocalSearch
	TmdbClient tmdb.Client
	Logger     *zap.SugaredLogger
}

func NewRunnerFromConfig(cfg Config, src Source, deps Dependencies) (Runner, error) {
	c := compiler{
		options: []compilerOption{
			compilerFeatures(defaultFeatures),
			exprEnvOption,
		},
		defaultLLMConfig: cfg.LLM,
		dependencies: dependencies{
			search:     deps.Search,
			tmdbClient: deps.TmdbClient,
			logger:     deps.Logger,
		},
	}

	r, err := c.Compile(src)
	if err != nil {
		return nil, err
	}

	return runnerSemaphore{
		runner:    r,
		semaphore: make(chan struct{}, cfg.Concurrency),
	}, nil
}

type Params struct {
	fx.In
	Config     Config
	TmdbConfig tmdb.Config
	Queries    utils.Lazy[*db.Queries]
	TmdbClient utils.Lazy[tmdb.Client]
	Logger     *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Compiler      utils.Lazy[Compiler]
	Source        utils.Lazy[Source]
	Runner        utils.Lazy[Runner]
	RebuildRunner func(Config) (Runner, error)
}

func New(params Params) Result {
	lc := utils.NewLazy(func() (Compiler, error) {
		q, err := params.Queries.Get()
		if err != nil {
			return nil, err
		}

		tmdbClient, err := params.TmdbClient.Get()
		if err != nil {
			return nil, err
		}

		return compiler{
			options: []compilerOption{
				compilerFeatures(defaultFeatures),
				func(src Source, ctx *compilerContext) error {
					if err := exprEnvOption(src, ctx); err != nil {
						params.Logger.Warnw("expression env setup failed", "error", err)
						return err
					}

					return nil
				},
			},
			defaultLLMConfig: params.Config.LLM,
			dependencies: dependencies{
				search: localSearchSemaphore{
					search:    localSearch{q: q, logger: params.Logger},
					semaphore: make(chan struct{}, 1),
				},
				tmdbClient: tmdbClient,
				logger:     params.Logger,
			},
		}, nil
	})
	lsrc := utils.NewLazy(func() (Source, error) {
		src, err := newSourceProvider(params.Config.Tmdb.Enabled).source()
		if err != nil {
			return Source{}, err
		}

		if _, ok := src.Workflows["default"]; !ok {
			return Source{}, fmt.Errorf("default workflow not found")
		}

		return src, nil
	})

	rebuildFn := func(cfg Config) (Runner, error) {
		// Build the source from the incoming config so tmdb.enabled changes
		// take effect on the rebuilt runner.
		src, err := newSourceProvider(cfg.Tmdb.Enabled).source()
		if err != nil {
			return nil, err
		}

		q, err := params.Queries.Get()
		if err != nil {
			return nil, err
		}

		tmdbClient, err := params.TmdbClient.Get()
		if err != nil {
			return nil, err
		}

		return NewRunnerFromConfig(cfg, src, Dependencies{
			Search: localSearchSemaphore{
				search:    localSearch{q: q, logger: params.Logger},
				semaphore: make(chan struct{}, 1),
			},
			TmdbClient: tmdbClient,
			Logger:     params.Logger,
		})
	}

	return Result{
		Compiler:      lc,
		Source:        lsrc,
		RebuildRunner: rebuildFn,
		Runner: utils.NewLazy(func() (Runner, error) {
			src, err := lsrc.Get()
			if err != nil {
				return nil, err
			}

			c, err := lc.Get()
			if err != nil {
				return nil, err
			}

			r, err := c.Compile(src)
			if err != nil {
				return nil, err
			}

			return runnerSemaphore{
				runner:    r,
				semaphore: make(chan struct{}, params.Config.Concurrency),
			}, nil
		}),
	}
}
