package logging

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Manager struct {
	dc         *dynamicCore
	level      *zap.AtomicLevel
	fileSyncer io.Closer
	mu         sync.Mutex
}

func (m *Manager) SubscribeToConfigManager(cm *configmgr.Manager) {
	if cm == nil {
		return
	}

	cm.Subscribe("log_manager",
		func(_ context.Context, snap *configmgr.Snapshot) error {
			return m.UpdateLogConfig(snap.Server.Log)
		}, configmgr.ApplyAsync)
}

func (m *Manager) UpdateLogConfig(cfg servercfg.LogConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.level.SetLevel(levelToZapLevel(cfg.ConsoleLevel))

	var (
		fileCore  zapcore.Core
		newSyncer io.Closer
	)

	if cfg.FileOutputLevel != "off" && cfg.FileRotator.Path != "" {
		ws := newFileRotator(cfg.FileRotator)

		var enc zapcore.Encoder
		if cfg.FileRotator.Format == "json" {
			enc = zapcore.NewJSONEncoder(jsonEncoderConfig)
		} else {
			enc = zapcore.NewConsoleEncoder(fileConsoleEncoderConfig)
		}

		fileCore = zapcore.NewCore(enc, ws, levelToZapLevel(cfg.FileOutputLevel))
		newSyncer = ws
	}

	cores := []zapcore.Core{
		zapcore.NewCore(zapcore.NewConsoleEncoder(consoleEncoderConfig), zapcore.AddSync(os.Stdout), m.level),
	}
	if fileCore != nil {
		cores = append(cores, fileCore)
	}

	m.dc.setCores(cores)

	if m.fileSyncer != nil {
		_ = m.fileSyncer.Close()
	}

	m.fileSyncer = newSyncer

	return nil
}
