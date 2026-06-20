package logging

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLevelToZapLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  zapcore.Level
	}{
		{"DEBUG", zapcore.DebugLevel},
		{"debug", zapcore.DebugLevel},
		{"  DEBUG  ", zapcore.DebugLevel},
		{"INFO", zapcore.InfoLevel},
		{"WARNING", zapcore.WarnLevel},
		{"ERROR", zapcore.ErrorLevel},
		{"CRITICAL", zapcore.DPanicLevel},
		{"ALERT", zapcore.PanicLevel},
		{"EMERGENCY", zapcore.FatalLevel},
		{"unknown", zapcore.WarnLevel},
		{"", zapcore.WarnLevel},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, levelToZapLevel(tt.input), "input: %q", tt.input)
	}
}

func TestLevelEncoder(t *testing.T) {
	t.Parallel()

	enc := levelEncoder()

	levels := []struct {
		level zapcore.Level
		want  string
	}{
		{zapcore.DebugLevel, "DEBUG"},
		{zapcore.InfoLevel, "INFO"},
		{zapcore.WarnLevel, "WARNING"},
		{zapcore.ErrorLevel, "ERROR"},
		{zapcore.DPanicLevel, "CRITICAL"},
		{zapcore.PanicLevel, "ALERT"},
		{zapcore.FatalLevel, "EMERGENCY"},
	}

	for _, l := range levels {
		buf := &testArrayEncoder{}
		enc(l.level, buf)
		assert.Equal(t, l.want, buf.values[0], "level: %v", l.level)
	}
}

type testArrayEncoder struct {
	values []string
}

func (e *testArrayEncoder) AppendString(v string)                      { e.values = append(e.values, v) }
func (*testArrayEncoder) AppendBool(_ bool)                            {}
func (*testArrayEncoder) AppendInt(_ int)                              {}
func (*testArrayEncoder) AppendInt8(_ int8)                            {}
func (*testArrayEncoder) AppendInt16(_ int16)                          {}
func (*testArrayEncoder) AppendInt32(_ int32)                          {}
func (*testArrayEncoder) AppendInt64(_ int64)                          {}
func (*testArrayEncoder) AppendUint(_ uint)                            {}
func (*testArrayEncoder) AppendUint8(_ uint8)                          {}
func (*testArrayEncoder) AppendUint16(_ uint16)                        {}
func (*testArrayEncoder) AppendUint32(_ uint32)                        {}
func (*testArrayEncoder) AppendUint64(_ uint64)                        {}
func (*testArrayEncoder) AppendUintptr(_ uintptr)                      {}
func (*testArrayEncoder) AppendFloat32(_ float32)                      {}
func (*testArrayEncoder) AppendFloat64(_ float64)                      {}
func (*testArrayEncoder) AppendByteString(_ []byte)                    {}
func (*testArrayEncoder) AppendArray(_ zapcore.ArrayMarshaler) error   { return nil }
func (*testArrayEncoder) AppendObject(_ zapcore.ObjectMarshaler) error { return nil }
func (*testArrayEncoder) AppendReflected(_ interface{}) error          { return nil }
func (*testArrayEncoder) AppendComplex64(_ complex64)                  {}
func (*testArrayEncoder) AppendComplex128(_ complex128)                {}
func (*testArrayEncoder) AppendDuration(_ time.Duration)               {}
func (*testArrayEncoder) AppendTime(_ time.Time)                       {}

func TestTimeEncoder(t *testing.T) {
	t.Parallel()

	enc := timeEncoder()
	buf := &testArrayEncoder{}
	now := time.Date(2024, 6, 15, 10, 30, 45, 123000000, time.UTC)
	enc(now, buf)
	assert.Equal(t, "2024-06-15T10:30:45.123Z", buf.values[0])
}

func TestNewDynamicCore(t *testing.T) {
	t.Parallel()

	dc := newDynamicCore(zapcore.NewNopCore())
	assert.NotNil(t, dc)
	assert.NotNil(t, dc.state)
	assert.Len(t, dc.state.cores, 1)
}

func TestDynamicCore_Enabled(t *testing.T) {
	t.Parallel()

	dc := newDynamicCore(newTestCore(false))
	assert.False(t, dc.Enabled(zapcore.InfoLevel))

	dc = newDynamicCore(newTestCore(true), newTestCore(false))
	assert.True(t, dc.Enabled(zapcore.InfoLevel))
}

func TestDynamicCore_With(t *testing.T) {
	t.Parallel()

	dc := newDynamicCore(zapcore.NewNopCore())
	field1 := zapcore.Field{Key: "key1", Type: zapcore.StringType, String: "val1"}
	field2 := zapcore.Field{Key: "key2", Type: zapcore.StringType, String: "val2"}

	dc2 := dc.With([]zapcore.Field{field1}).(*dynamicCore)
	dc3 := dc2.With([]zapcore.Field{field2}).(*dynamicCore)

	assert.Empty(t, dc.fields)
	assert.Len(t, dc2.fields, 1)
	assert.Len(t, dc3.fields, 2)
	assert.Equal(t, "val1", dc3.fields[0].String)
	assert.Equal(t, "val2", dc3.fields[1].String)
}

func TestDynamicCore_Sync(t *testing.T) {
	t.Parallel()

	var syncCalled bool

	dc := newDynamicCore(newSyncCore(func() error {
		syncCalled = true
		return nil
	}))

	err := dc.Sync()
	require.NoError(t, err)
	assert.True(t, syncCalled)
}

func TestDynamicCore_SyncError(t *testing.T) {
	t.Parallel()

	dc := newDynamicCore(newSyncCore(func() error {
		return assert.AnError
	}))

	err := dc.Sync()
	assert.Error(t, err)
}

func TestDynamicCore_Write(t *testing.T) {
	t.Parallel()

	var writeCalled bool

	dc := newDynamicCore(newWriteCore(func(_ zapcore.Entry, _ []zapcore.Field) error {
		writeCalled = true
		return nil
	}))

	err := dc.Write(zapcore.Entry{}, nil)
	require.NoError(t, err)
	assert.True(t, writeCalled)
}

func TestDynamicCore_WriteError(t *testing.T) {
	t.Parallel()

	dc := newDynamicCore(newWriteCore(func(_ zapcore.Entry, _ []zapcore.Field) error {
		return assert.AnError
	}))

	err := dc.Write(zapcore.Entry{}, nil)
	assert.Error(t, err)
}

func TestDynamicCore_WriteWithFields(t *testing.T) {
	t.Parallel()

	var captured []zapcore.Field

	dc := newDynamicCore(newWriteCore(func(_ zapcore.Entry, fs []zapcore.Field) error {
		captured = fs
		return nil
	}))
	dc = dc.With([]zapcore.Field{{Key: "base", Type: zapcore.StringType, String: "base"}}).(*dynamicCore)

	err := dc.Write(zapcore.Entry{}, []zapcore.Field{{Key: "extra", Type: zapcore.StringType, String: "extra"}})
	require.NoError(t, err)
	require.Len(t, captured, 2)
	assert.Equal(t, "base", captured[0].String)
	assert.Equal(t, "extra", captured[1].String)
}

func TestDynamicCore_SetCores(t *testing.T) {
	t.Parallel()

	dc := newDynamicCore(zapcore.NewNopCore())
	assert.Len(t, dc.state.cores, 1)

	dc.setCores([]zapcore.Core{zapcore.NewNopCore(), zapcore.NewNopCore()})
	assert.Len(t, dc.state.cores, 2)
}

func TestNewFileRotator(t *testing.T) {
	t.Parallel()

	cfg := servercfg.FileRotatorConfig{
		Path:       "/tmp/logs",
		MaxBackups: 7,
		Format:     "json",
	}

	fr := newFileRotator(cfg)
	assert.NotNil(t, fr)
	assert.Equal(t, "/tmp/logs", fr.path)
	assert.Equal(t, defaultBaseName, fr.baseName)
	assert.Equal(t, 7, fr.maxBackups)
	assert.False(t, fr.pathCreated)
	assert.Nil(t, fr.file)
	assert.False(t, fr.closed)
}

func TestNewFilePath(t *testing.T) {
	t.Parallel()

	fr := &fileRotator{path: "/var/log", baseName: "hexmagnet"}
	now := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	filePath := fr.newFilePath(now)
	assert.Equal(t, "/var/log/hexmagnet.2024-06-15.log", filePath)
}

func TestFileRotator_ShouldRotate_NilFile(t *testing.T) {
	t.Parallel()

	fr := &fileRotator{}
	assert.True(t, fr.shouldRotate())
}

func TestFileRotator_ShouldRotate_SameDate(t *testing.T) {
	t.Parallel()

	fr := &fileRotator{
		file:     &fileRotatorFile{},
		fileDate: time.Now().Format(timeFormat),
	}
	assert.False(t, fr.shouldRotate())
}

func TestFileRotator_ShouldRotate_DifferentDate(t *testing.T) {
	t.Parallel()

	fr := &fileRotator{
		file:     &fileRotatorFile{},
		fileDate: "2020-01-01",
	}
	assert.True(t, fr.shouldRotate())
}

func TestFileRotator_WriteClosed(t *testing.T) {
	t.Parallel()

	fr := &fileRotator{closed: true}
	n, err := fr.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
}

func TestFileRotator_WriteCreatesDir(t *testing.T) {
	t.Parallel()

	logDir := filepath.Join(t.TempDir(), "subdir", "logs")
	fr := &fileRotator{
		path:       logDir,
		baseName:   defaultBaseName,
		maxBackups: 2,
	}

	n, err := fr.Write([]byte("test\n"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	_, statErr := os.Stat(logDir)
	require.NoError(t, statErr, "directory should exist")

	_ = fr.Close()
}

func TestFileRotator_SyncWithoutFile(t *testing.T) {
	t.Parallel()

	fr := &fileRotator{}
	err := fr.Sync()
	require.NoError(t, err)
}

func TestFileRotator_SyncWithFile(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fp, openErr := newFileRotatorFile(filepath.Join(logDir, "hexmagnet.test.log"))
	require.NoError(t, openErr)

	fr := &fileRotator{file: fp}
	err := fr.Sync()
	require.NoError(t, err)
}

func TestFileRotator_Close(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fp, openErr := newFileRotatorFile(filepath.Join(logDir, "hexmagnet.test.log"))
	require.NoError(t, openErr)

	fr := &fileRotator{file: fp}
	err := fr.Close()
	require.NoError(t, err)
	assert.True(t, fr.closed)
}

func TestFileRotator_CloseWithoutFile(t *testing.T) {
	t.Parallel()

	fr := &fileRotator{}
	err := fr.Close()
	require.NoError(t, err)
	assert.True(t, fr.closed)
}

func TestFileRotator_CloseIdempotent(t *testing.T) {
	t.Parallel()

	fr := &fileRotator{}
	err1 := fr.Close()
	err2 := fr.Close()

	require.NoError(t, err1)
	require.NoError(t, err2)
}

func TestFileRotator_Rotate(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fr := &fileRotator{
		path:       logDir,
		baseName:   defaultBaseName,
		maxBackups: 5,
	}

	err := fr.rotate()
	require.NoError(t, err)
	assert.NotNil(t, fr.file)
	assert.Equal(t, time.Now().Format(timeFormat), fr.fileDate)
}

func TestFileRotator_RotateClosesOldFile(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fp, openErr := newFileRotatorFile(filepath.Join(logDir, "hexmagnet.old.log"))
	require.NoError(t, openErr)

	fr := &fileRotator{
		path:       logDir,
		baseName:   defaultBaseName,
		maxBackups: 5,
		file:       fp,
		fileDate:   "2020-01-01",
	}

	err := fr.rotate()
	require.NoError(t, err)
	assert.NotNil(t, fr.file)
	assert.NotEqual(t, fp, fr.file)
}

func TestFileRotator_PruneBackups(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fr := &fileRotator{
		path:       logDir,
		baseName:   defaultBaseName,
		maxBackups: 2,
	}

	now := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	for i := range 5 {
		date := time.Date(2024, 6, 10+i, 0, 0, 0, 0, time.UTC)
		f, createErr := os.Create(filepath.Join(logDir, defaultBaseName+"."+date.Format(timeFormat)+".log"))
		require.NoError(t, createErr)

		_ = f.Close()
	}

	err := fr.pruneBackups(now)
	require.NoError(t, err)

	files, readErr := os.ReadDir(logDir)
	require.NoError(t, readErr)
	assert.LessOrEqual(t, len(files), 2)
}

func TestFileRotator_PruneBackupsLessThanMax(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fr := &fileRotator{
		path:       logDir,
		baseName:   defaultBaseName,
		maxBackups: 10,
	}

	now := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	for i := range 3 {
		date := time.Date(2024, 6, 10+i, 0, 0, 0, 0, time.UTC)
		f, createErr := os.Create(filepath.Join(logDir, defaultBaseName+"."+date.Format(timeFormat)+".log"))
		require.NoError(t, createErr)

		_ = f.Close()
	}

	err := fr.pruneBackups(now)
	require.NoError(t, err)

	files, readErr := os.ReadDir(logDir)
	require.NoError(t, readErr)
	assert.Len(t, files, 3)
}

func TestFileRotator_PruneBackupsSkipsNonMatchingFiles(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fr := &fileRotator{
		path:       logDir,
		baseName:   defaultBaseName,
		maxBackups: 0,
	}

	now := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	_, _ = os.Create(filepath.Join(logDir, "other.log"))
	_, _ = os.Create(filepath.Join(logDir, defaultBaseName+".2024-06-10.txt"))
	_, _ = os.Create(filepath.Join(logDir, "subdir"))

	err := fr.pruneBackups(now)
	require.NoError(t, err)

	files, readErr := os.ReadDir(logDir)
	require.NoError(t, readErr)
	assert.Len(t, files, 3)
}

func TestFileRotator_WriteIntegration(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fr := newFileRotator(servercfg.FileRotatorConfig{
		Path:       logDir,
		MaxBackups: 2,
	})

	n, err := fr.Write([]byte("integration test\n"))
	require.NoError(t, err)
	assert.Equal(t, 17, n)

	closeErr := fr.Close()
	require.NoError(t, closeErr)
}

func TestManager_UpdateLogConfig(t *testing.T) {
	t.Parallel()

	m := newTestManager()
	require.NotNil(t, m)

	cfg := servercfg.LogConfig{
		ConsoleLevel: "debug",
	}

	err := m.UpdateLogConfig(cfg)
	require.NoError(t, err)
	assert.Equal(t, zapcore.DebugLevel, m.level.Level())
}

func TestManager_UpdateLogConfigWithFileOutput(t *testing.T) {
	t.Parallel()

	m := newTestManager()
	logDir := t.TempDir()

	cfg := servercfg.LogConfig{
		ConsoleLevel:    "info",
		FileOutputLevel: "debug",
		FileRotator: servercfg.FileRotatorConfig{
			Path:       logDir,
			MaxBackups: 3,
		},
	}

	err := m.UpdateLogConfig(cfg)
	require.NoError(t, err)
	assert.Equal(t, zapcore.InfoLevel, m.level.Level())
}

func TestManager_UpdateLogConfigFileOutputOff(t *testing.T) {
	t.Parallel()

	m := newTestManager()

	cfg := servercfg.LogConfig{
		ConsoleLevel:    "error",
		FileOutputLevel: "off",
	}

	err := m.UpdateLogConfig(cfg)
	require.NoError(t, err)
	assert.Equal(t, zapcore.ErrorLevel, m.level.Level())
}

func TestManager_UpdateLogConfigReplacesFileSyncer(t *testing.T) {
	t.Parallel()

	m := newTestManager()
	logDir1 := t.TempDir()
	logDir2 := t.TempDir()

	err := m.UpdateLogConfig(servercfg.LogConfig{
		ConsoleLevel:    "info",
		FileOutputLevel: "debug",
		FileRotator: servercfg.FileRotatorConfig{
			Path:       logDir1,
			MaxBackups: 3,
		},
	})
	require.NoError(t, err)

	err = m.UpdateLogConfig(servercfg.LogConfig{
		ConsoleLevel:    "info",
		FileOutputLevel: "debug",
		FileRotator: servercfg.FileRotatorConfig{
			Path:       logDir2,
			MaxBackups: 3,
		},
	})
	require.NoError(t, err)
}

func TestFileRotatorFile_Flush(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fp, err := newFileRotatorFile(filepath.Join(logDir, "test.log"))
	require.NoError(t, err)

	n, writeErr := fp.Write([]byte("data\n"))
	require.NoError(t, writeErr)
	assert.Equal(t, 5, n)

	flushErr := fp.Flush()
	require.NoError(t, flushErr)

	closeErr := fp.Close()
	require.NoError(t, closeErr)
}

func TestFileRotatorFile_CloseFlushes(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fp, err := newFileRotatorFile(filepath.Join(logDir, "test.log"))
	require.NoError(t, err)

	_, writeErr := fp.Write([]byte("data\n"))
	require.NoError(t, writeErr)

	closeErr := fp.Close()
	require.NoError(t, closeErr)
}

func TestFileRotator_WriteCreateDirError(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	blockFile := filepath.Join(logDir, "block")
	err := os.WriteFile(blockFile, []byte("block"), 0o644)
	require.NoError(t, err)

	fr := &fileRotator{
		path:       filepath.Join(blockFile, "subdir"),
		baseName:   defaultBaseName,
		maxBackups: 2,
	}

	_, err = fr.Write([]byte("test"))
	assert.Error(t, err)
}

func TestFileRotator_ConcurrentWrite(t *testing.T) {
	t.Parallel()

	logDir := t.TempDir()
	fr := newFileRotator(servercfg.FileRotatorConfig{
		Path:       logDir,
		MaxBackups: 5,
	})

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, writeErr := fr.Write([]byte("concurrent\n"))
			assert.NoError(t, writeErr)
		}()
	}

	wg.Wait()

	closeErr := fr.Close()
	require.NoError(t, closeErr)
}

func TestManager_UpdateLogConfigJSONFormat(t *testing.T) {
	t.Parallel()

	m := newTestManager()
	logDir := t.TempDir()

	err := m.UpdateLogConfig(servercfg.LogConfig{
		ConsoleLevel:    "info",
		FileOutputLevel: "debug",
		FileRotator: servercfg.FileRotatorConfig{
			Path:       logDir,
			Format:     "json",
			MaxBackups: 3,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, zapcore.InfoLevel, m.level.Level())
}

type testCore struct {
	zapcore.Core
	enabled bool
}

func newTestCore(enabled bool) *testCore {
	return &testCore{Core: zapcore.NewNopCore(), enabled: enabled}
}

func (c *testCore) Enabled(_ zapcore.Level) bool {
	return c.enabled
}

type syncCore struct {
	zapcore.Core
	syncFn func() error
}

func newSyncCore(fn func() error) *syncCore {
	return &syncCore{Core: zapcore.NewNopCore(), syncFn: fn}
}

func (c *syncCore) Sync() error {
	return c.syncFn()
}

type writeCore struct {
	zapcore.Core
	writeFn func(zapcore.Entry, []zapcore.Field) error
}

func newWriteCore(fn func(zapcore.Entry, []zapcore.Field) error) *writeCore {
	return &writeCore{Core: zapcore.NewNopCore(), writeFn: fn}
}

func (c *writeCore) Write(entry zapcore.Entry, fs []zapcore.Field) error {
	return c.writeFn(entry, fs)
}

func newTestManager() *Manager {
	al := zap.NewAtomicLevel()

	return &Manager{
		dc:    newDynamicCore(),
		level: &al,
	}
}
