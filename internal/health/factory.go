package health

import (
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

		return NewChecker(params.Options...), nil
	})

	return Result{
		Checker:          lChecker,
		HTTPServerOption: handlerBuilder{lChecker},
	}
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
