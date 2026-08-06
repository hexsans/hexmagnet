package worker

import (
	"context"
	"sync"
	"testing"

	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestNewWorker_Key(t *testing.T) {
	t.Parallel()

	w := NewWorker("test", fx.Hook{})
	assert.Equal(t, "test", w.Key())
}

func TestNewWorker_Defaults(t *testing.T) {
	t.Parallel()

	w := NewWorker("test", fx.Hook{})
	assert.False(t, w.Enabled())
	assert.False(t, w.Started())
}

func TestWorker_SetEnabled(t *testing.T) {
	t.Parallel()

	w := NewWorker("test", fx.Hook{}).(*worker)
	w.setEnabled(true)
	assert.True(t, w.Enabled())

	w.setEnabled(false)
	assert.False(t, w.Enabled())
}

func TestWorker_SetStarted(t *testing.T) {
	t.Parallel()

	w := NewWorker("test", fx.Hook{}).(*worker)
	w.setStarted(true)
	assert.True(t, w.Started())

	w.setStarted(false)
	assert.False(t, w.Started())
}

func TestNewRegistry_Empty(t *testing.T) {
	t.Parallel()

	result, err := NewRegistry(RegistryParams{
		Logger: testutil.NewTestLogger(),
	})

	require.NoError(t, err)
	assert.Empty(t, result.Registry.Workers())
}

func TestRegistry_Workers(t *testing.T) {
	t.Parallel()

	result, err := NewRegistry(RegistryParams{
		Workers: []Worker{
			NewWorker("a", fx.Hook{}),
			NewWorker("b", fx.Hook{}),
		},
		Logger: testutil.NewTestLogger(),
	})

	require.NoError(t, err)

	workers := result.Registry.Workers()
	require.Len(t, workers, 2)
	assert.Equal(t, "a", workers[0].Key())
	assert.Equal(t, "b", workers[1].Key())
}

func TestRegistry_EnableDisable(t *testing.T) {
	t.Parallel()

	result, err := NewRegistry(RegistryParams{
		Workers: []Worker{
			NewWorker("test", fx.Hook{}),
		},
		Logger: testutil.NewTestLogger(),
	})
	require.NoError(t, err)

	err = result.Registry.Enable("test")
	require.NoError(t, err)

	workers := result.Registry.Workers()
	require.Len(t, workers, 1)
	assert.True(t, workers[0].Enabled())

	err = result.Registry.Disable("test")
	require.NoError(t, err)

	workers = result.Registry.Workers()
	assert.False(t, workers[0].Enabled())
}

func TestRegistry_Enable_NotFound(t *testing.T) {
	t.Parallel()

	result, err := NewRegistry(RegistryParams{
		Logger: testutil.NewTestLogger(),
	})
	require.NoError(t, err)

	err = result.Registry.Enable("nonexistent")
	assert.ErrorContains(t, err, "not found")
}

func TestRegistry_Disable_NotFound(t *testing.T) {
	t.Parallel()

	result, err := NewRegistry(RegistryParams{
		Logger: testutil.NewTestLogger(),
	})
	require.NoError(t, err)

	err = result.Registry.Disable("nonexistent")
	assert.ErrorContains(t, err, "not found")
}

func TestRegistry_EnableAll_DisableAll(t *testing.T) {
	t.Parallel()

	result, err := NewRegistry(RegistryParams{
		Workers: []Worker{
			NewWorker("a", fx.Hook{}),
			NewWorker("b", fx.Hook{}),
		},
		Logger: testutil.NewTestLogger(),
	})
	require.NoError(t, err)

	result.Registry.EnableAll()

	for _, w := range result.Registry.Workers() {
		assert.True(t, w.Enabled())
	}

	result.Registry.DisableAll()

	for _, w := range result.Registry.Workers() {
		assert.False(t, w.Enabled())
	}
}

func TestRegistry_Start(t *testing.T) {
	t.Parallel()

	var started bool

	w := NewWorker("test", fx.Hook{
		OnStart: func(_ context.Context) error {
			started = true
			return nil
		},
	})

	result, err := NewRegistry(RegistryParams{
		Workers: []Worker{w},
		Logger:  testutil.NewTestLogger(),
	})
	require.NoError(t, err)

	result.Registry.EnableAll()

	ctx := context.Background()
	err = result.Registry.Start(ctx)
	require.NoError(t, err)
	assert.True(t, started)

	workers := result.Registry.Workers()
	assert.True(t, workers[0].Started())
}

func TestRegistry_Start_Twice(t *testing.T) {
	t.Parallel()

	w := NewWorker("test", fx.Hook{
		OnStart: func(_ context.Context) error {
			return nil
		},
	})

	result, err := NewRegistry(RegistryParams{
		Workers: []Worker{w},
		Logger:  testutil.NewTestLogger(),
	})
	require.NoError(t, err)

	result.Registry.EnableAll()

	ctx := context.Background()
	err = result.Registry.Start(ctx)
	require.NoError(t, err)

	err = result.Registry.Start(ctx)
	assert.ErrorContains(t, err, "already started")
}

func TestRegistry_Start_NoEnabledWorkers(t *testing.T) {
	t.Parallel()

	result, err := NewRegistry(RegistryParams{
		Workers: []Worker{
			NewWorker("test", fx.Hook{}),
		},
		Logger: testutil.NewTestLogger(),
	})
	require.NoError(t, err)

	err = result.Registry.Start(context.Background())
	assert.ErrorIs(t, err, ErrNoWorkersEnabled)
}

func TestRegistry_Stop(t *testing.T) {
	t.Parallel()

	var stopped bool

	w := NewWorker("test", fx.Hook{
		OnStart: func(_ context.Context) error { return nil },
		OnStop:  func(_ context.Context) error { stopped = true; return nil },
	})

	result, err := NewRegistry(RegistryParams{
		Workers: []Worker{w},
		Logger:  testutil.NewTestLogger(),
	})
	require.NoError(t, err)

	ctx := context.Background()

	result.Registry.EnableAll()
	_ = result.Registry.Start(ctx)
	err = result.Registry.Stop(ctx)
	require.NoError(t, err)
	assert.True(t, stopped)
}

func TestRegistry_Workers_Sorted(t *testing.T) {
	t.Parallel()

	result, err := NewRegistry(RegistryParams{
		Workers: []Worker{
			NewWorker("z", fx.Hook{}),
			NewWorker("a", fx.Hook{}),
			NewWorker("m", fx.Hook{}),
		},
		Logger: testutil.NewTestLogger(),
	})
	require.NoError(t, err)

	workers := result.Registry.Workers()
	require.Len(t, workers, 3)
	assert.Equal(t, "a", workers[0].Key())
	assert.Equal(t, "m", workers[1].Key())
	assert.Equal(t, "z", workers[2].Key())
}

func TestGoRecover(t *testing.T) {
	t.Parallel()

	logger := testutil.NewTestLogger()

	var wg sync.WaitGroup
	wg.Add(1)

	GoRecover(logger, "test", func() {
		defer wg.Done()
	})

	wg.Wait()
}

func TestGoRecover_Panic(t *testing.T) {
	t.Parallel()

	logger := testutil.NewTestLogger()

	GoRecover(logger, "panic_test", func() {
		panic("test panic")
	})
}
