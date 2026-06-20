package db

import (
	"testing"

	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFx(t *testing.T) {
	t.Parallel()

	p := Params{}
	r := NewFx(p)
	assert.NotNil(t, r.Queries)
}

func TestNewQueries(t *testing.T) {
	t.Parallel()

	q := NewQueries(nil)
	assert.NotNil(t, q)
	assert.NotNil(t, q.Queries)
	assert.Nil(t, q.pool)
}

func TestToProtocolID(t *testing.T) {
	t.Parallel()

	id := ToProtocolID(testutil.ValidHash())
	assert.Equal(t, testutil.ValidHash(), id.String())
}

func TestToProtocolID_PanicsOnInvalidLength(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		ToProtocolID("tooshort")
	})
}

func TestToProtocolID_PanicsOnNonHex(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		ToProtocolID("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	})
}

func TestFromProtocolID(t *testing.T) {
	t.Parallel()

	id, err := protocol.ParseID(testutil.ValidHash())
	require.NoError(t, err)
	assert.Equal(t, testutil.ValidHash(), FromProtocolID(id))
}

func TestFromUintToInt32Ptr(t *testing.T) {
	t.Parallel()

	ptr := FromUintToInt32Ptr(42)
	require.NotNil(t, ptr)
	assert.Equal(t, int32(42), *ptr)

	ptr = FromUintToInt32Ptr(0)
	require.NotNil(t, ptr)
	assert.Equal(t, int32(0), *ptr)
}

func TestNewNullStringFromPtr_Nil(t *testing.T) {
	t.Parallel()

	ns := model.NewNullStringFromPtr(nil)
	assert.False(t, ns.Valid)
	assert.Empty(t, ns.String)
}

func TestNewNullStringFromPtr_Value(t *testing.T) {
	t.Parallel()

	s := "hello"
	ns := model.NewNullStringFromPtr(&s)
	assert.True(t, ns.Valid)
	assert.Equal(t, "hello", ns.String)
}

func TestNewNullStringFromPtr_EmptyString(t *testing.T) {
	t.Parallel()

	s := ""
	ns := model.NewNullStringFromPtr(&s)
	assert.True(t, ns.Valid)
	assert.Empty(t, ns.String)
}

func TestNullString_Ptr_Invalid(t *testing.T) {
	t.Parallel()

	ptr := model.NullString{}.Ptr()
	assert.Nil(t, ptr)
}

func TestNullString_Ptr_Valid(t *testing.T) {
	t.Parallel()

	ns := model.NewNullString("world")
	ptr := ns.Ptr()
	require.NotNil(t, ptr)
	assert.Equal(t, "world", *ptr)
}

func TestToNullUint_Nil(t *testing.T) {
	t.Parallel()

	nu := toNullUint(nil)
	assert.False(t, nu.Valid)
	assert.Equal(t, uint(0), nu.Uint)
}

func TestToNullUint_Value(t *testing.T) {
	t.Parallel()

	v := int32(99)
	nu := toNullUint(&v)
	assert.True(t, nu.Valid)
	assert.Equal(t, uint(99), nu.Uint)
}

func TestToNullUint_Zero(t *testing.T) {
	t.Parallel()

	v := int32(0)
	nu := toNullUint(&v)
	assert.True(t, nu.Valid)
	assert.Equal(t, uint(0), nu.Uint)
}

func TestFromNullUint_Invalid(t *testing.T) {
	t.Parallel()

	ptr := fromNullUint(model.NullUint{})
	assert.Nil(t, ptr)
}

func TestFromNullUint_Valid(t *testing.T) {
	t.Parallel()

	nu := model.NewNullUint(77)
	ptr := fromNullUint(nu)
	require.NotNil(t, ptr)
	assert.Equal(t, int32(77), *ptr)
}

func TestFromNullUint_Zero(t *testing.T) {
	t.Parallel()

	nu := model.NewNullUint(0)
	ptr := fromNullUint(nu)
	require.NotNil(t, ptr)
	assert.Equal(t, int32(0), *ptr)
}

func TestFromNullUint64_Invalid(t *testing.T) {
	t.Parallel()

	ptr := fromNullUint64(model.NullUint{})
	assert.Nil(t, ptr)
}

func TestFromNullUint64_Valid(t *testing.T) {
	t.Parallel()

	nu := model.NewNullUint(12345)
	ptr := fromNullUint64(nu)
	require.NotNil(t, ptr)
	assert.Equal(t, int64(12345), *ptr)
}

func TestNullBool_Ptr_Invalid(t *testing.T) {
	t.Parallel()

	ptr := model.NullBool{}.Ptr()
	assert.Nil(t, ptr)
}

func TestNullBool_Ptr_Valid(t *testing.T) {
	t.Parallel()

	nb := model.NewNullBool(true)
	ptr := nb.Ptr()
	require.NotNil(t, ptr)
	assert.True(t, *ptr)

	nb = model.NewNullBool(false)
	ptr = nb.Ptr()
	require.NotNil(t, ptr)
	assert.False(t, *ptr)
}

func TestFromNullFloat32_Invalid(t *testing.T) {
	t.Parallel()

	ptr := fromNullFloat32(model.NullFloat32{})
	assert.Nil(t, ptr)
}

func TestFromNullFloat32_Valid(t *testing.T) {
	t.Parallel()

	nf := model.NewNullFloat32(3.14)
	ptr := fromNullFloat32(nf)
	require.NotNil(t, ptr)
	assert.InDelta(t, float64(3.140000104904175), *ptr, 0.0001)
}

func TestFromNullFloat32_Zero(t *testing.T) {
	t.Parallel()

	nf := model.NewNullFloat32(0)
	ptr := fromNullFloat32(nf)
	require.NotNil(t, ptr)
	assert.InDelta(t, float64(0), *ptr, 0.0001)
}
