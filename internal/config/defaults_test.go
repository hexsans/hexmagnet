package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructToYamlMap_Scalars(t *testing.T) {
	t.Parallel()

	type Scalars struct {
		Name  string
		Age   int
		Ratio float64
		Flag  bool
	}

	v := Scalars{Name: "hello", Age: 42, Ratio: 3.14, Flag: true}
	result, err := structToYamlMap(reflect.ValueOf(v))
	require.NoError(t, err)

	assert.Equal(t, "hello", result["name"])
	assert.Equal(t, 42, result["age"])
	assert.InEpsilon(t, 3.14, result["ratio"], 1e-9)
	assert.Equal(t, true, result["flag"])
}

func TestStructToYamlMap_Nested(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Value string
	}

	type Outer struct {
		InnerField Inner
	}

	v := Outer{InnerField: Inner{Value: "nested"}}
	result, err := structToYamlMap(reflect.ValueOf(v))
	require.NoError(t, err)

	inner, ok := result["inner_field"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "nested", inner["value"])
}

func TestStructToYamlMap_Duration(t *testing.T) {
	t.Parallel()

	type Config struct {
		Timeout time.Duration
	}

	v := Config{Timeout: 5 * time.Second}
	result, err := structToYamlMap(reflect.ValueOf(v))
	require.NoError(t, err)

	assert.Equal(t, "5s", result["timeout"])
}

func TestStructToYamlMap_StructSlice(t *testing.T) {
	t.Parallel()

	type Item struct {
		Name  string
		Value int
	}

	type Container struct {
		Items []Item
	}

	v := Container{Items: []Item{{Name: "a", Value: 1}, {Name: "b", Value: 2}}}
	result, err := structToYamlMap(reflect.ValueOf(v))
	require.NoError(t, err)

	items, ok := result["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)

	first := items[0].(map[string]any)
	assert.Equal(t, "a", first["name"])
	assert.Equal(t, 1, first["value"])

	second := items[1].(map[string]any)
	assert.Equal(t, "b", second["name"])
	assert.Equal(t, 2, second["value"])
}

func TestStructToYamlMap_StringSlice(t *testing.T) {
	t.Parallel()

	type Config struct {
		Tags []string
	}

	v := Config{Tags: []string{"go", "test"}}
	result, err := structToYamlMap(reflect.ValueOf(v))
	require.NoError(t, err)

	assert.Equal(t, []string{"go", "test"}, result["tags"])
}

func TestStructToYamlMap_PointerField(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Val string
	}

	type Outer struct {
		Ptr *Inner
	}

	v := Outer{Ptr: &Inner{Val: "ptr-value"}}
	result, err := structToYamlMap(reflect.ValueOf(v))
	require.NoError(t, err)

	inner, ok := result["ptr"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ptr-value", inner["val"])
}

func TestStructToYamlMap_ZeroValues(t *testing.T) {
	t.Parallel()

	type Config struct {
		Name    string
		Count   int
		Enabled bool
	}

	v := Config{}
	result, err := structToYamlMap(reflect.ValueOf(v))
	require.NoError(t, err)

	assert.Empty(t, result["name"])
	assert.Equal(t, 0, result["count"])
	assert.Equal(t, false, result["enabled"])
}

func TestStructToYamlMap_NonStructError(t *testing.T) {
	t.Parallel()

	_, err := structToYamlMap(reflect.ValueOf("not a struct"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected struct")
}

func TestStructToYamlMap_NilSlice(t *testing.T) {
	t.Parallel()

	type Config struct {
		Items []string
	}

	v := Config{}
	result, err := structToYamlMap(reflect.ValueOf(v))
	require.NoError(t, err)

	assert.Nil(t, result["items"])
}

func TestPlaceAtPath_SingleKey(t *testing.T) {
	t.Parallel()

	m := make(map[string]any)
	placeAtPath(m, []string{"key"}, "value")

	assert.Equal(t, "value", m["key"])
}

func TestPlaceAtPath_NestedKeys(t *testing.T) {
	t.Parallel()

	m := make(map[string]any)
	placeAtPath(m, []string{"a", "b", "c"}, 42)

	inner := m["a"].(map[string]any)
	inner2 := inner["b"].(map[string]any)
	assert.Equal(t, 42, inner2["c"])
}

func TestPlaceAtPath_Overwrite(t *testing.T) {
	t.Parallel()

	m := make(map[string]any)
	placeAtPath(m, []string{"a", "b"}, "first")
	placeAtPath(m, []string{"a", "b"}, "second")

	assert.Equal(t, "second", m["a"].(map[string]any)["b"])
}

func TestBuildDefaultConfigMap(t *testing.T) {
	t.Parallel()

	type Server struct {
		Port int
	}

	type Database struct {
		Host string
	}

	result, err := buildDefaultConfigMap([]SpecEntry{
		{Key: "server", DefaultValue: Server{Port: 3333}},
		{Key: "storage.postgres", DefaultValue: Database{Host: "localhost"}},
	})
	require.NoError(t, err)

	server, ok := result["server"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 3333, server["port"])

	storage, ok := result["storage"].(map[string]any)
	require.True(t, ok)
	postgres, ok := storage["postgres"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "localhost", postgres["host"])
}

func TestBuildDefaultConfigMap_NilDefault(t *testing.T) {
	t.Parallel()

	type Server struct {
		Port int
	}

	result, err := buildDefaultConfigMap([]SpecEntry{
		{Key: "server", DefaultValue: Server{Port: 3333}},
	})
	require.NoError(t, err)

	server := result["server"].(map[string]any)
	assert.Equal(t, 3333, server["port"])
}
