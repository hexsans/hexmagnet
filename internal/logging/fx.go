package logging

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewModule() fx.Option {
	return fx.Module(
		"logging",
		fx.Provide(New),
	)
}

func WithLogger() fx.Option {
	return fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
		l := &fxevent.ZapLogger{Logger: log.Named("fx")}
		l.UseLogLevel(zapcore.DebugLevel)

		return l
	})
}
