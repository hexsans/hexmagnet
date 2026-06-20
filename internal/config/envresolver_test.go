package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvResolver_Resolve_Unset(t *testing.T) {
	t.Parallel()

	r := NewEnv(map[string]string{}, WithPriority(10))
	got, ok, err := r.Resolve([]string{"my_key"}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestEnvResolver_Resolve_String(t *testing.T) {
	t.Parallel()

	r := NewEnv(map[string]string{"MY_KEY": "hello"}, WithPriority(10))
	got, ok, err := r.Resolve([]string{"my_key"}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "hello", got)
}

func TestEnvResolver_Resolve_Int(t *testing.T) {
	t.Parallel()

	r := NewEnv(map[string]string{"MY_KEY": "42"}, WithPriority(10))
	got, ok, err := r.Resolve([]string{"my_key"}, reflect.TypeOf(int(0)))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 42, got)
}

func TestEnvResolver_Resolve_Duration(t *testing.T) {
	t.Parallel()

	r := NewEnv(map[string]string{"MY_KEY": "30s"}, WithPriority(10))
	got, ok, err := r.Resolve([]string{"my_key"}, reflect.TypeOf(time.Duration(0)))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 30*time.Second, got)
}

func TestEnvResolver_Resolve_CoercionError(t *testing.T) {
	t.Parallel()

	r := NewEnv(map[string]string{"MY_KEY": "not_a_number"}, WithPriority(10))
	_, ok, err := r.Resolve([]string{"my_key"}, reflect.TypeOf(int(0)))
	require.Error(t, err)
	assert.True(t, ok)
	assert.Contains(t, err.Error(), "error coercing")
	assert.Contains(t, err.Error(), "MY_KEY")
}

func TestEnvResolver_Resolve_NestedPath(t *testing.T) {
	t.Parallel()

	r := NewEnv(map[string]string{"PARENT_CHILD_KEY": "nested_val"}, WithPriority(10))
	got, ok, err := r.Resolve([]string{"parent", "child", "key"}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "nested_val", got)
}
