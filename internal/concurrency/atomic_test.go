package concurrency

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAtomicValue_GetSet(t *testing.T) {
	t.Parallel()

	var v AtomicValue[int]
	v.Set(42)
	assert.Equal(t, 42, v.Get())

	v.Set(100)
	assert.Equal(t, 100, v.Get())
}

func TestAtomicValue_ZeroValue(t *testing.T) {
	t.Parallel()

	var v AtomicValue[string]
	assert.Empty(t, v.Get())
}

func TestAtomicValue_Update(t *testing.T) {
	t.Parallel()

	var v AtomicValue[int]
	v.Set(10)

	result := v.Update(func(val int) int {
		return val + 5
	})

	assert.Equal(t, 15, result)
	assert.Equal(t, 15, v.Get())
}

func TestAtomicValue_Update_Chain(t *testing.T) {
	t.Parallel()

	var v AtomicValue[string]
	v.Set("hello")

	v.Update(func(val string) string {
		return val + " world"
	})
	assert.Equal(t, "hello world", v.Get())

	v.Update(func(val string) string {
		return val + "!"
	})
	assert.Equal(t, "hello world!", v.Get())
}

func TestAtomicValue_Concurrency(t *testing.T) {
	t.Parallel()

	var v AtomicValue[int]

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			v.Update(func(val int) int {
				return val + n
			})
		}(i)
	}

	wg.Wait()

	expected := 0
	for i := range 100 {
		expected += i
	}

	assert.Equal(t, expected, v.Get())
}
