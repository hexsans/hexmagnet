package queue

import (
	"context"
	"fmt"

	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewModule(cfg Config) fx.Option {
	return fx.Module(
		"queue",
		fx.Provide(
			func(logger *zap.SugaredLogger) *Runtime {
				return NewRuntime(cfg, logger)
			},
			func(r *Runtime) Producer {
				return r.DynamicProducer()
			},
			func(r *Runtime) ConsumerMaker {
				return r.DynamicConsumerMaker()
			},
			func(r *Runtime) utils.Lazy[Manager] {
				return &managerLazy{runtime: r}
			},
			NewHealthCheckFx,
		),
		fx.Invoke(func(lc fx.Lifecycle, r *Runtime, _ *zap.SugaredLogger, pgxPool utils.Lazy[*pgxpool.Pool]) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					if pool, err := pgxPool.Get(); err == nil {
						r.memoryQueue.SetPool(pool)
						r.memoryQueue.Recover(ctx)
					}

					if r.ActiveBackend.Get() == backendMemory {
						r.memoryQueueActive = true
						go r.startMemoryPersistLoop(ctx)
					}

					return nil
				},
				OnStop: func(ctx context.Context) error {
					_ = r.CloseMemoryQueue(ctx)
					return nil
				},
			})
		}),
	)
}

type managerLazy struct {
	runtime *Runtime
}

func (l *managerLazy) Get() (Manager, error) {
	mgr := l.runtime.Manager.Get()
	if mgr == nil {
		return nil, fmt.Errorf("queue manager not initialized")
	}

	return mgr, nil
}

func (*managerLazy) Decorate(_ func(Manager) (Manager, error)) {
}

func (l *managerLazy) IfInitialized(fn func(Manager) error) error {
	if mgr := l.runtime.Manager.Get(); mgr != nil {
		return fn(mgr)
	}

	return nil
}
