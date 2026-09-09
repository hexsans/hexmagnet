package utils

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClampInt32(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int32(0), ClampInt32(0))
	assert.Equal(t, int32(42), ClampInt32(42))
	assert.Equal(t, int32(-42), ClampInt32(-42))
	assert.Equal(t, int32(math.MaxInt32), ClampInt32(int64(math.MaxInt64)))
	assert.Equal(t, int32(math.MinInt32), ClampInt32(int64(math.MinInt64)))
	assert.Equal(t, int32(math.MaxInt32), ClampInt32(uint64(math.MaxUint64)))
	assert.Equal(t, int32(math.MaxInt32), ClampInt32(uint64(math.MaxInt64)))
	assert.Equal(t, int32(10), ClampInt32(uint(10)))
	assert.Equal(t, int32(math.MaxInt32), ClampInt32(uint(math.MaxUint64)))
}

func TestClampUint16(t *testing.T) {
	t.Parallel()

	assert.Equal(t, uint16(0), ClampUint16(0))
	assert.Equal(t, uint16(8080), ClampUint16(8080))
	assert.Equal(t, uint16(0), ClampUint16(-1))
	assert.Equal(t, uint16(0), ClampUint16(int64(-100)))
	assert.Equal(t, uint16(65535), ClampUint16(70000))
	assert.Equal(t, uint16(65535), ClampUint16(int64(math.MaxInt64)))
	assert.Equal(t, uint16(65535), ClampUint16(uint64(math.MaxUint64)))
}

func TestClampUint8(t *testing.T) {
	t.Parallel()

	assert.Equal(t, uint8(0), ClampUint8(0))
	assert.Equal(t, uint8(31), ClampUint8(31))
	assert.Equal(t, uint8(0), ClampUint8(-1))
	assert.Equal(t, uint8(255), ClampUint8(256))
	assert.Equal(t, uint8(255), ClampUint8(int64(math.MaxInt64)))
	assert.Equal(t, uint8(255), ClampUint8(uint64(math.MaxUint64)))
}

func TestClampUint32(t *testing.T) {
	t.Parallel()

	assert.Equal(t, uint32(0), ClampUint32(0))
	assert.Equal(t, uint32(5), ClampUint32(5))
	assert.Equal(t, uint32(0), ClampUint32(-1))
	assert.Equal(t, uint32(math.MaxUint32), ClampUint32(int64(math.MaxInt64)))
	assert.Equal(t, uint32(math.MaxUint32), ClampUint32(uint64(math.MaxUint64)))
}
