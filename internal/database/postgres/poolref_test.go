package postgres

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestPoolRef_New_Get(t *testing.T) {
	t.Parallel()

	pool := new(pgxpool.Pool)
	ref := NewPoolRef(pool)

	assert.Same(t, pool, ref.Get())
}

func TestPoolRef_New_Nil(t *testing.T) {
	t.Parallel()

	ref := NewPoolRef(nil)
	assert.Nil(t, ref.Get())
}

func TestPoolRef_Replace_Swaps(t *testing.T) {
	t.Parallel()

	pool1 := new(pgxpool.Pool)
	pool2 := new(pgxpool.Pool)
	ref := NewPoolRef(pool1)

	old := ref.Replace(pool2)

	assert.Same(t, pool1, old, "Replace returns old pool")
	assert.Same(t, pool2, ref.Get(), "Get returns new pool after Replace")
}

func TestPoolRef_Replace_ReturnsPrevious(t *testing.T) {
	t.Parallel()

	pool1, pool2, pool3 := new(pgxpool.Pool), new(pgxpool.Pool), new(pgxpool.Pool)
	ref := NewPoolRef(pool1)

	ref.Replace(pool2)
	old := ref.Replace(pool3)

	assert.Same(t, pool2, old, "Replace returns the pool that was current before the call")
	assert.Same(t, pool3, ref.Get())
}

func TestPoolRef_Replace_ConcurrentSafe(t *testing.T) {
	t.Parallel()

	pool := new(pgxpool.Pool)
	ref := NewPoolRef(pool)

	done := make(chan bool, 2)

	writer := func() {
		for range 100 {
			ref.Replace(new(pgxpool.Pool))
		}

		done <- true
	}

	reader := func() {
		for range 100 {
			_ = ref.Get()
		}

		done <- true
	}

	go writer()
	go reader()

	<-done
	<-done

	assert.NotNil(t, ref.Get())
}
