package protocol

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInt160FromByteArray(t *testing.T) {
	t.Parallel()

	bits := [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	got := NewInt160FromByteArray(bits)
	assert.Equal(t, bits, got.AsByteArray())
}

func TestInt160String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value [20]byte
		want  string
	}{
		{"all zeros", [20]byte{}, "0000000000000000000000000000000000000000"},
		{"leading bytes", [20]byte{0xDE, 0xAD}, "dead000000000000000000000000000000000000"},
		{
			"all max", func() (a [20]byte) {
				for i := range a {
					a[i] = 0xFF
				}

				return
			}(),
			"ffffffffffffffffffffffffffffffffffffffff",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			i := NewInt160FromByteArray(tt.value)
			assert.Equal(t, tt.want, i.String())
		})
	}
}

func TestInt160Bytes(t *testing.T) {
	t.Parallel()

	bits := [20]byte{0xFF, 0xFE, 0xFD, 0xFC}
	i := NewInt160FromByteArray(bits)
	assert.Equal(t, bits[:], i.Bytes())
	assert.Len(t, i.Bytes(), 20)
}

func TestInt160AsByteArray(t *testing.T) {
	t.Parallel()

	bits := [20]byte{0xFF, 0xFE, 0xFD, 0xFC}
	i := NewInt160FromByteArray(bits)
	assert.Equal(t, bits, i.AsByteArray())
}

func TestInt160ByteString(t *testing.T) {
	t.Parallel()

	bits := [20]byte{'h', 'e', 'l', 'l', 'o'}
	i := NewInt160FromByteArray(bits)
	want := "hello" + string([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	assert.Equal(t, want, i.ByteString())
}

func TestInt160BitLen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		i    Int160
		want int
	}{
		{"zero", Int160{}, 0},
		{"byte 0 = 0x80", Int160{bits: [20]byte{0x80}}, 160},
		{"byte 0 = 0x01", Int160{bits: [20]byte{0x01}}, 153},
		{"byte 0 = 0x0F", Int160{bits: [20]byte{0x0F}}, 156},
		{"byte 0 = 0x40", Int160{bits: [20]byte{0x40}}, 159},
		{"byte 19 = 0x01 (LSB)", Int160{bits: [20]byte{19: 0x01}}, 1},
		{"byte 19 = 0x80 (LSB)", Int160{bits: [20]byte{19: 0x80}}, 8},
		{"all max", Int160{}.WithMax(), 160},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.i.BitLen())
		})
	}
}

func TestInt160GetBit(t *testing.T) {
	t.Parallel()

	i := Int160{bits: [20]byte{0b10000000, 0b10000000}}
	assert.True(t, i.GetBit(0))
	assert.False(t, i.GetBit(1))
	assert.False(t, i.GetBit(7))
	assert.True(t, i.GetBit(8))
	assert.False(t, i.GetBit(15))
}

func TestInt160WithBit(t *testing.T) {
	t.Parallel()
	t.Run("set bit 0", func(t *testing.T) {
		t.Parallel()

		i := Int160{}.WithBit(0, true)
		assert.Equal(t, byte(0x80), i.bits[0])
	})
	t.Run("set bit 7", func(t *testing.T) {
		t.Parallel()

		i := Int160{}.WithBit(7, true)
		assert.Equal(t, byte(0x01), i.bits[0])
	})
	t.Run("set bit 8 (byte boundary)", func(t *testing.T) {
		t.Parallel()

		i := Int160{}.WithBit(8, true)
		assert.Equal(t, byte(0x80), i.bits[1])
	})
	t.Run("set bit 159 (last bit)", func(t *testing.T) {
		t.Parallel()

		i := Int160{}.WithBit(159, true)
		assert.Equal(t, byte(0x01), i.bits[19])
	})
	t.Run("clear bit", func(t *testing.T) {
		t.Parallel()

		i := Int160{bits: [20]byte{0x80}}.WithBit(0, false)
		assert.Equal(t, byte(0x00), i.bits[0])
	})
	t.Run("set and get round-trip", func(t *testing.T) {
		t.Parallel()

		for _, idx := range []int{0, 1, 7, 8, 15, 63, 64, 127, 128, 159} {
			i := Int160{}.WithBit(idx, true)
			assert.True(t, i.GetBit(idx), "bit %d should be set", idx)
		}
	})
}

func TestInt160Cmp(t *testing.T) {
	t.Parallel()

	zero := Int160{}
	one := Int160{bits: [20]byte{19: 1}}
	two := Int160{bits: [20]byte{19: 2}}
	highByte := Int160{bits: [20]byte{0: 1}}

	tests := []struct {
		name string
		a, b Int160
		want int
	}{
		{"equal", zero, zero, 0},
		{"equal non-zero", one, one, 0},
		{"less (last byte)", zero, one, -1},
		{"greater (last byte)", one, zero, 1},
		{"less (last byte 1<2)", one, two, -1},
		{"greater (last byte 2>1)", two, one, 1},
		{"first byte dominates", highByte, Int160{bits: [20]byte{19: 0xFF}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.a.Cmp(tt.b))
		})
	}
}

func TestInt160WithMax(t *testing.T) {
	t.Parallel()

	i := Int160{}.WithMax()

	expected := [20]byte{}
	for b := range expected {
		expected[b] = math.MaxUint8
	}

	assert.Equal(t, expected, i.AsByteArray())
	assert.Equal(t, 160, i.BitLen())
	assert.True(t, i.GetBit(0))
	assert.True(t, i.GetBit(159))
}

func TestInt160IsZero(t *testing.T) {
	t.Parallel()
	assert.True(t, (Int160{}).IsZero())
	assert.False(t, (Int160{bits: [20]byte{1}}).IsZero())
}

func TestInt160Xor(t *testing.T) {
	t.Parallel()

	a := Int160{bits: [20]byte{0xFF}}
	b := Int160{bits: [20]byte{0x0F}}
	result := (Int160{}).Xor(a, b)
	assert.Equal(t, byte(0xF0), result.bits[0])
}

func TestInt160Distance(t *testing.T) {
	t.Parallel()

	a := Int160{bits: [20]byte{0xFF}}
	b := Int160{bits: [20]byte{0x0F}}
	d := a.Distance(b)
	assert.Equal(t, byte(0xF0), d.bits[0])
}
