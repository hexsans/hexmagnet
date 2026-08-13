package config

import (
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoerceStringValue_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		target  any
		want    any
		wantErr bool
	}{
		{input: "hello", target: string(""), want: "hello"},
		{input: "true", target: bool(false), want: true},
		{input: "false", target: bool(false), want: false},
		{input: "1", target: bool(false), want: true},
		{input: "0", target: bool(false), want: false},
		{input: "invalid", target: bool(false), wantErr: true},
		{input: "42", target: int(0), want: 42},
		{input: "127", target: int8(0), want: int8(127)},
		{input: "1000", target: int16(0), want: int16(1000)},
		{input: "100000", target: int32(0), want: int32(100000)},
		{input: "10000000000", target: int64(0), want: int64(10000000000)},
		{input: "42", target: uint(0), want: uint(42)},
		{input: "255", target: uint8(0), want: uint8(255)},
		{input: "1000", target: uint16(0), want: uint16(1000)},
		{input: "100000", target: uint32(0), want: uint32(100000)},
		{input: "10000000000", target: uint64(0), want: uint64(10000000000)},
		{input: "3.14", target: float32(0), want: float32(3.14)},
		{input: "3.1415926535", target: float64(0), want: 3.1415926535},
		{input: "5s", target: time.Duration(0), want: time.Second * 5},
		{input: "-1", target: uint(0), wantErr: true},
		{input: "not_a_number", target: int(0), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := coerceStringValue(tt.input, reflect.TypeOf(tt.target))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCoerceStringValue_Uint(t *testing.T) {
	t.Parallel()

	maxUintStr := strconv.FormatUint(uint64(^uint(0)), 10)
	got, err := coerceStringValue(maxUintStr, reflect.TypeOf(uint(0)))
	require.NoError(t, err)
	assert.Equal(t, ^uint(0), got)

	overflow := "4294967296" // 2^32
	if strconv.IntSize == 64 {
		overflow = "18446744073709551616" // 2^64
	}

	_, err = coerceStringValue(overflow, reflect.TypeOf(uint(0)))
	require.Error(t, err)
}

func TestCoerceStringValue_Slice(t *testing.T) {
	t.Parallel()

	got, err := coerceStringValue("1,2,3", reflect.TypeOf([]int{}))
	require.NoError(t, err)

	values, ok := got.([]any)
	require.True(t, ok)
	assert.Equal(t, []any{1, 2, 3}, values)

	got2, err := coerceStringValue("a,b,c", reflect.TypeOf([]string{}))
	require.NoError(t, err)

	values2, ok := got2.([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"a", "b", "c"}, values2)
}

func TestCoerceStringValue_InvalidType(t *testing.T) {
	t.Parallel()

	_, err := coerceStringValue("test", reflect.TypeOf(map[string]string{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}
