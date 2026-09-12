package health

import (
	"context"
	"slices"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hexsans/hexmagnet/internal/httpserver"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	Options []CheckerOption `group:"health_check_options"`
	Logger  *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Checker          utils.Lazy[Checker]
	HTTPServerOption httpserver.Option `group:"http_server_options"`
}

func New(params Params) Result {
	lChecker := utils.NewLazy[Checker](func() (Checker, error) {
		defer func() {
			if r := recover(); r != nil {
				params.Logger.Errorw("health checker initialization panicked",
					"panic", r,
				)
				panic(r)
			}
		}()

		options := append(slices.Clone(params.Options), WithStatusChangeListener(statusLogger(params.Logger)))

		return NewChecker(options...), nil
	})

	return Result{
		Checker:          lChecker,
		HTTPServerOption: handlerBuilder{lChecker},
	}
}

// statusLogger logs service availability transitions. It only emits on state
// changes, so a failing dependency does not flood the log while it stays down.
func statusLogger(logger *zap.SugaredLogger) func(context.Context, CheckerState) {
	var (
		mu       sync.Mutex
		previous AvailabilityStatus
		seen     bool
	)

	return func(_ context.Context, state CheckerState) {
		mu.Lock()

		wasDown := seen && previous == StatusDown
		previous = state.Status
		seen = true

		mu.Unlock()

		switch state.Status {
		case StatusUp:
			if wasDown {
				logger.Infow("service health recovered")
			}
		case StatusDown:
			logger.Errorw("service health degraded", "failing", failingChecks(state))
		default:
			logger.Warnw("service health unknown", "failing", failingChecks(state))
		}
	}
}

func failingChecks(state CheckerState) []string {
	failing := make([]string, 0, len(state.CheckState))

	for name, check := range state.CheckState {
		if check.Status != StatusDown {
			continue
		}

		msg := name
		if check.Result != nil {
			msg = name + ": " + check.Result.Error()
		}

		failing = append(failing, msg)
	}

	slices.Sort(failing)

	return failing
}

type handlerBuilder struct {
	Checker utils.Lazy[Checker]
}

func (handlerBuilder) Key() string {
	return "health"
}

func (b handlerBuilder) Apply(e *gin.Engine) error {
	checker, err := b.Checker.Get()
	if err != nil {
		return err
	}

	handler := NewHandler(checker)

	e.GET("/status", func(c *gin.Context) {
		handler(c.Writer, c.Request)
	})

	return nil
}
