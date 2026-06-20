package utils

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLazy_Get_CallsFnOnce(t *testing.T) {
	t.Parallel()

	var callCount int

	fn := func() (int, error) {
		callCount++
		return 42, nil
	}

	l := NewLazy(fn)

	assert.Equal(t, 0, callCount, "fn should not be called before first Get")

	v, err := l.Get()
	require.NoError(t, err)
	assert.Equal(t, 42, v)
	assert.Equal(t, 1, callCount)

	for range 10 {
		v, err = l.Get()
		require.NoError(t, err)
		assert.Equal(t, 42, v)
	}

	assert.Equal(t, 1, callCount, "fn should be called only once")
}

func TestLazy_Get_ReturnsError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("init error")
	fn := func() (int, error) {
		return 0, expectedErr
	}

	l := NewLazy(fn)
	v, err := l.Get()
	require.ErrorIs(t, err, expectedErr)
	assert.Zero(t, v)

	_, err = l.Get()
	assert.ErrorIs(t, err, expectedErr, "subsequent calls should still return the error")
}

func TestLazy_Get_Concurrent(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	fn := func() (int, error) {
		callCount.Add(1)
		return 42, nil
	}

	l := NewLazy(fn)

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			v, err := l.Get()
			assert.NoError(t, err)
			assert.Equal(t, 42, v)
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(1), callCount.Load())
}

func TestLazy_Decorate(t *testing.T) {
	t.Parallel()

	l := NewLazy(func() (int, error) { return 5, nil })
	l.Decorate(func(v int) (int, error) {
		return v * 2, nil
	})

	v, err := l.Get()
	require.NoError(t, err)
	assert.Equal(t, 10, v)
}

func TestLazy_Decorate_Chained(t *testing.T) {
	t.Parallel()

	l := NewLazy(func() (int, error) { return 1, nil })
	l.Decorate(func(v int) (int, error) { return v + 2, nil })
	l.Decorate(func(v int) (int, error) { return v * 10, nil })

	v, err := l.Get()
	require.NoError(t, err)
	assert.Equal(t, 30, v) // ((1) + 2) * 10
}

func TestLazy_Decorate_SkipsOnBaseError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("base error")
	l := NewLazy(func() (int, error) { return 0, expectedErr })
	decorateCalled := false

	l.Decorate(func(v int) (int, error) {
		decorateCalled = true
		return v, nil
	})

	v, err := l.Get()
	require.ErrorIs(t, err, expectedErr)
	assert.Zero(t, v)
	assert.False(t, decorateCalled, "decorate fn should not be called when base fn fails")
}

func TestLazy_Decorate_PropagatesDecoratorError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("decorator error")
	l := NewLazy(func() (int, error) { return 5, nil })
	l.Decorate(func(_ int) (int, error) {
		return 0, expectedErr
	})

	v, err := l.Get()
	require.ErrorIs(t, err, expectedErr)
	assert.Zero(t, v)
}

func TestLazy_IfInitialized_RunsWhenInitialized(t *testing.T) {
	t.Parallel()

	l := NewLazy(func() (string, error) { return "hello", nil })
	_, err := l.Get()
	require.NoError(t, err)

	var result string

	err = l.IfInitialized(func(v string) error {
		result = v
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestLazy_IfInitialized_NotCalledWhenNotInitialized(t *testing.T) {
	t.Parallel()

	l := NewLazy(func() (string, error) { return "hello", nil })
	called := false
	err := l.IfInitialized(func(_ string) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	assert.False(t, called)
}

func TestLazy_IfInitialized_NotCalledWhenInitFailed(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("fail")
	l := NewLazy(func() (string, error) { return "", expectedErr })
	_, err := l.Get()
	require.Error(t, err)

	called := false
	err = l.IfInitialized(func(_ string) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	assert.False(t, called)
}

func TestLazy_IfInitialized_PropagatesCallbackError(t *testing.T) {
	t.Parallel()

	cbErr := errors.New("cb error")
	l := NewLazy(func() (string, error) { return "val", nil })
	_, err := l.Get()
	require.NoError(t, err)

	err = l.IfInitialized(func(_ string) error {
		return cbErr
	})
	assert.ErrorIs(t, err, cbErr)
}
