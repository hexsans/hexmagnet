// Package supervisor runs queue consumers with a managed lifecycle: it
// retries consumer creation, recreates the consumer when the queue backend
// changes, and reacts to optional config events.
package supervisor

import (
	"context"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/backoff"
	"github.com/hexsans/hexmagnet/internal/queue"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var retryDelay = backoff.Config{Base: time.Second, Factor: 1, Max: time.Second}

// BackendChanges is the subset of queue.Runtime the supervisor needs.
type BackendChanges interface {
	SubscribeBackendChanges() (<-chan struct{}, func())
}

// Trigger is an optional additional event source. When an event arrives,
// OnEvent is called; returning true stops and recreates the consumer.
type Trigger struct {
	Subscribe func() (<-chan struct{}, func())
	OnEvent   func(ctx context.Context) bool
}

type Params struct {
	Maker   queue.ConsumerMaker
	Runtime BackendChanges
	Topic   string
	GroupID string
	Handler queue.MessageHandler
	// Prepare, when set, is called once before the first consumer is created.
	// It may resolve dependencies lazily; an error stops the supervisor.
	Prepare func(ctx context.Context) (queue.MessageHandler, error)
	Trigger *Trigger
	Logger  *zap.SugaredLogger

	retryDelay *backoff.Config
}

type Supervisor struct {
	params Params

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(params Params) *Supervisor {
	return &Supervisor{params: params}
}

// Hook returns the fx lifecycle hook that runs the supervisor on its own
// context. The run context is detached from the lifecycle start context and
// canceled in OnStop, so shutdown does not depend on the start context being
// canceled afterwards.
func (s *Supervisor) Hook() fx.Hook {
	return fx.Hook{
		OnStart: func(ctx context.Context) error {
			runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))

			s.mu.Lock()
			s.cancel = cancel
			s.mu.Unlock()

			s.wg.Add(1)

			go func() {
				defer s.wg.Done()

				s.run(runCtx)
			}()

			return nil
		},
		OnStop: func(context.Context) error {
			s.mu.Lock()
			cancel := s.cancel
			s.mu.Unlock()

			if cancel != nil {
				cancel()
			}

			s.wg.Wait()

			return nil
		},
	}
}

func (s *Supervisor) run(ctx context.Context) {
	backendCh, unsubscribe := s.params.Runtime.SubscribeBackendChanges()
	defer unsubscribe()

	handler := s.params.Handler
	if s.params.Prepare != nil {
		prepared, err := s.params.Prepare(ctx)
		if err != nil {
			s.params.Logger.Errorw("failed to prepare consumer", "error", err)

			return
		}

		handler = prepared
	}

	var extra <-chan struct{}

	onExtra := func(context.Context) bool { return false }

	if trigger := s.params.Trigger; trigger != nil {
		events, unsubscribeTrigger := trigger.Subscribe()
		defer unsubscribeTrigger()

		extra = events

		if trigger.OnEvent != nil {
			onExtra = trigger.OnEvent
		}
	}

	delay := retryDelay
	if s.params.retryDelay != nil {
		delay = *s.params.retryDelay
	}

	for {
		consumer, err := s.params.Maker.NewConsumer(s.params.Topic, s.params.GroupID, handler, s.params.Logger)
		if err != nil {
			s.params.Logger.Errorw("failed to create consumer", "error", err)

			if delay.Wait(ctx, 1) != nil {
				return
			}

			continue
		}

		if err := consumer.Start(ctx); err != nil {
			s.params.Logger.Errorw("failed to start consumer", "error", err)

			if delay.Wait(ctx, 1) != nil {
				return
			}

			continue
		}

		s.params.Logger.Infow("consumer started", "topic", s.params.Topic)

		restart := false

		for !restart {
			select {
			case <-backendCh:
				s.params.Logger.Infow("backend changed, restarting consumer")

				restart = true
			case <-extra:
				restart = onExtra(ctx)
			case <-ctx.Done():
				s.stop(ctx, consumer)

				return
			}
		}

		s.stop(ctx, consumer)
	}
}

func (s *Supervisor) stop(ctx context.Context, consumer queue.Consumer) {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := consumer.Stop(stopCtx); err != nil {
		s.params.Logger.Errorw("failed to stop consumer", "error", err)
	}
}
