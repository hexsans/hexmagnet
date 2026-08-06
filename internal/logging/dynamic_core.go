package logging

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

type coreState struct {
	mu    sync.RWMutex
	cores []zapcore.Core
}

type dynamicCore struct {
	state  *coreState
	fields []zapcore.Field
}

func newDynamicCore(cores ...zapcore.Core) *dynamicCore {
	return &dynamicCore{
		state: &coreState{cores: cores},
	}
}

func (dc *dynamicCore) Enabled(level zapcore.Level) bool {
	dc.state.mu.RLock()
	defer dc.state.mu.RUnlock()

	for _, c := range dc.state.cores {
		if c.Enabled(level) {
			return true
		}
	}

	return false
}

func (dc *dynamicCore) With(fs []zapcore.Field) zapcore.Core {
	newFields := make([]zapcore.Field, len(dc.fields)+len(fs))
	copy(newFields, dc.fields)
	copy(newFields[len(dc.fields):], fs)

	return &dynamicCore{
		state:  dc.state,
		fields: newFields,
	}
}

func (dc *dynamicCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	dc.state.mu.RLock()
	cores := dc.state.cores
	dc.state.mu.RUnlock()

	for _, c := range cores {
		ce = c.Check(entry, ce)
	}

	return ce
}

func (dc *dynamicCore) Write(entry zapcore.Entry, fs []zapcore.Field) error {
	dc.state.mu.RLock()
	cores := dc.state.cores
	dc.state.mu.RUnlock()
	allFields := make([]zapcore.Field, len(dc.fields)+len(fs))
	copy(allFields, dc.fields)
	copy(allFields[len(dc.fields):], fs)

	for _, c := range cores {
		if err := c.Write(entry, allFields); err != nil {
			return err
		}
	}

	return nil
}

func (dc *dynamicCore) Sync() error {
	dc.state.mu.RLock()
	cores := dc.state.cores
	dc.state.mu.RUnlock()

	for _, c := range cores {
		if err := c.Sync(); err != nil {
			return err
		}
	}

	return nil
}

func (dc *dynamicCore) setCores(cores []zapcore.Core) {
	dc.state.mu.Lock()
	defer dc.state.mu.Unlock()

	dc.state.cores = cores
}
