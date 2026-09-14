package retryqueue

import (
	"context"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SchedulerParams struct {
	fx.In
	Queue  *Queue
	Logger *zap.SugaredLogger
}

type SchedulerResult struct {
	fx.Out
	Worker worker.Worker `group:"workers"`
}

func NewScheduler(p SchedulerParams) SchedulerResult {
	logger := p.Logger.Named("retry_queue.scheduler")

	var (
		wg     sync.WaitGroup
		cancel context.CancelFunc
	)

	return SchedulerResult{
		Worker: worker.NewWorker("retry_queue_scheduler", fx.Hook{
			OnStart: func(startCtx context.Context) error {
				// The lifecycle start context may be cancelled or short-lived
				// depending on how the app is run; the scheduler must run on
				// its own context that OnStop can cancel.
				runCtx, runCancel := context.WithCancel(context.WithoutCancel(startCtx))
				cancel = runCancel

				wg.Add(1)

				go runScheduler(runCtx, p.Queue, logger, &wg)

				return nil
			},
			OnStop: func(_ context.Context) error {
				if cancel != nil {
					cancel()
				}

				wg.Wait()

				return nil
			},
		}),
	}
}

func runScheduler(ctx context.Context, q *Queue, logger *zap.SugaredLogger, wg *sync.WaitGroup) {
	defer wg.Done()

	timer := time.NewTimer(scanInterval(q.cfg.Get()))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
		case <-ctx.Done():
			return
		}

		func() {
			defer utils.Recover(logger, "retry queue scheduler panicked")

			q.dispatchDue(ctx)
		}()

		timer.Reset(scanInterval(q.cfg.Get()))
	}
}

func scanInterval(cfg Config) time.Duration {
	if cfg.ScanInterval <= 0 {
		return 30 * time.Second
	}

	return cfg.ScanInterval
}
