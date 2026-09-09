package utils

import "math"

// Integer covers the signed and unsigned integer types that conversions in
// this package clamp from.
type Integer interface {
	~int | ~int64 | ~uint | ~uint64
}

// ClampInt32 converts an integer of architecture-dependent size to int32,
// saturating to [math.MinInt32, math.MaxInt32] instead of wrapping.
func ClampInt32[T Integer](v T) int32 {
	switch x := any(v).(type) {
	case int:
		if x > math.MaxInt32 {
			return math.MaxInt32
		}
		if x < math.MinInt32 {
			return math.MinInt32
		}
		return int32(x)
	case int64:
		if x > math.MaxInt32 {
			return math.MaxInt32
		}
		if x < math.MinInt32 {
			return math.MinInt32
		}
		return int32(x)
	case uint:
		if x > math.MaxInt32 {
			return math.MaxInt32
		}
		return int32(x)
	case uint64:
		if x > math.MaxInt32 {
			return math.MaxInt32
		}
		return int32(x)
	default:
		panic("unsupported integer type")
	}
}

// ClampUint16 converts an integer of architecture-dependent size to uint16,
// saturating to [0, math.MaxUint16] instead of wrapping.
func ClampUint16[T Integer](v T) uint16 {
	if v > T(math.MaxUint16) {
		return math.MaxUint16
	}

	if int64(v) < 0 {
		return 0
	}

	return uint16(v)
}

// ClampUint8 converts an integer of architecture-dependent size to uint8,
// saturating to [0, math.MaxUint8] instead of wrapping.
func ClampUint8[T Integer](v T) uint8 {
	if v > T(math.MaxUint8) {
		return math.MaxUint8
	}

	if int64(v) < 0 {
		return 0
	}

	return uint8(v)
}

// ClampUint32 converts an integer of architecture-dependent size to uint32,
// saturating to [0, math.MaxUint32] instead of wrapping.
func ClampUint32[T Integer](v T) uint32 {
	if v > T(math.MaxUint32) {
		return math.MaxUint32
	}

	if int64(v) < 0 {
		return 0
	}

	return uint32(v)
}
