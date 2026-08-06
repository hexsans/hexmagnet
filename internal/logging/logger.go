package logging

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Params struct {
	fx.In
	Config        servercfg.Config
	ConfigManager *configmgr.Manager `optional:"true"`
}

type Result struct {
	fx.Out
	Logger      *zap.Logger
	Sugar       *zap.SugaredLogger
	AtomicLevel *zap.AtomicLevel
	Manager     *Manager
	AppHook     fx.Hook `group:"app_hooks"`
}

func New(params Params) Result {
	logCfg := params.Config.Log

	opts := []zap.Option{
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCaller(),
	}

	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(levelToZapLevel(logCfg.ConsoleLevel))

	dc := newDynamicCore()

	cores, fileSyncer, err := buildCores(logCfg, &atomicLevel)
	if err != nil {
		panic(err)
	}

	dc.setCores(cores)

	manager := &Manager{
		dc:         dc,
		level:      &atomicLevel,
		fileSyncer: fileSyncer,
	}

	l := zap.New(dc, opts...)

	manager.SubscribeToConfigManager(params.ConfigManager)

	return Result{
		Logger:      l,
		Sugar:       l.Sugar(),
		AtomicLevel: &atomicLevel,
		Manager:     manager,
		AppHook: fx.Hook{
			OnStop: func(context.Context) error {
				return l.Sync()
			},
		},
	}
}

func buildCores(logCfg servercfg.LogConfig, atomicLevel *zap.AtomicLevel) ([]zapcore.Core, io.Closer, error) {
	cores := []zapcore.Core{
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEncoderConfig),
			zapcore.AddSync(os.Stdout),
			atomicLevel,
		),
	}

	var fileSyncer io.Closer

	if logCfg.FileOutputLevel != "off" && logCfg.FileRotator.Path != "" {
		if err := os.MkdirAll(logCfg.FileRotator.Path, 0o755); err != nil {
			return nil, nil, fmt.Errorf("create log directory %s: %w", logCfg.FileRotator.Path, err)
		}

		ws := newFileRotator(logCfg.FileRotator)
		fileSyncer = ws

		var fileEncoder zapcore.Encoder
		if logCfg.FileRotator.Format == "json" {
			fileEncoder = zapcore.NewJSONEncoder(jsonEncoderConfig)
		} else {
			fileEncoder = zapcore.NewConsoleEncoder(fileConsoleEncoderConfig)
		}

		cores = append(cores, zapcore.NewCore(
			fileEncoder,
			ws,
			levelToZapLevel(logCfg.FileOutputLevel),
		))
	}

	return cores, fileSyncer, nil
}
