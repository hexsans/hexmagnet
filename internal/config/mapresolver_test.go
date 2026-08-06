package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapResolver_NewMap(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{"key": "value"}
	r := NewMap(m, val, WithPriority(5))
	assert.Equal(t, 5, r.Priority())
	assert.Equal(t, "map", r.Key())

	got, ok, err := r.Resolve([]string{"key"}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "value", got)
}

func TestMapResolver_Resolve_MissingKey(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{"existing": "val"}
	r := NewMap(m, val)

	got, ok, err := r.Resolve([]string{"nonexistent"}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestMapResolver_Resolve_NestedPath(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{
		"parent": map[string]any{
			"child": "nested",
		},
	}
	r := NewMap(m, val)

	got, ok, err := r.Resolve([]string{"parent", "child"}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "nested", got)
}

func TestMapResolver_Resolve_CoerceString(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{"count": "42"}
	r := NewMap(m, val)

	got, ok, err := r.Resolve([]string{"count"}, reflect.TypeOf(int(0)))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 42, got)
}

func TestMapResolver_Resolve_CoerceDuration(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{"timeout": "10s"}
	r := NewMap(m, val)

	got, ok, err := r.Resolve([]string{"timeout"}, reflect.TypeOf(time.Duration(0)))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 10*time.Second, got)
}

func TestMapResolver_Resolve_NonStringValue(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{"count": 99}
	r := NewMap(m, val)

	got, ok, err := r.Resolve([]string{"count"}, reflect.TypeOf(int(0)))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 99, got)
}

func TestMapResolver_Resolve_NilValue(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{"key": nil}
	r := NewMap(m, val)

	got, ok, err := r.Resolve([]string{"key"}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestMapResolver_Resolve_NilIntermediate(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{"parent": nil}
	r := NewMap(m, val)

	got, ok, err := r.Resolve([]string{"parent", "child"}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestMapResolver_Resolve_WrongTypeIntermediate(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{"parent": "string"}
	r := NewMap(m, val)

	_, ok, err := r.Resolve([]string{"parent", "child"}, reflect.TypeOf(""))
	require.Error(t, err)
	assert.True(t, ok)
	assert.Contains(t, err.Error(), "expected map[string]interface{}")
}

func TestMapResolver_Resolve_CoercionError(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{"key": "not_a_bool"}
	r := NewMap(m, val)

	_, ok, err := r.Resolve([]string{"key"}, reflect.TypeOf(false))
	require.Error(t, err)
	assert.True(t, ok)
	assert.Contains(t, err.Error(), "error coercing")
}

func TestMapResolver_Resolve_EmptyPath(t *testing.T) {
	t.Parallel()

	val := validator.New()
	m := map[string]any{"key": "val"}
	r := NewMap(m, val)

	got, ok, err := r.Resolve([]string{}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, m, got)
}
