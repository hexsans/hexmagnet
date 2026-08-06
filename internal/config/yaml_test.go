package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewFromYamlFile_ExistingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := map[string]any{"key": "value", "count": 42}
	yamlBytes, err := yaml.Marshal(data)
	require.NoError(t, err)
	err = os.WriteFile(path, yamlBytes, 0o644)
	require.NoError(t, err)

	val := validator.New()
	r, err := NewFromYamlFile(path, false, val)
	require.NoError(t, err)
	assert.NotNil(t, r)

	got, ok, err := r.Resolve([]string{"key"}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "value", got)

	gotCount, ok, err := r.Resolve([]string{"count"}, reflect.TypeOf(int(0)))
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 42, gotCount)
}

func TestNewFromYamlFile_IgnoreMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.yaml")

	val := validator.New()
	r, err := NewFromYamlFile(path, true, val)
	require.NoError(t, err)
	assert.NotNil(t, r)

	got, ok, err := r.Resolve([]string{"anything"}, reflect.TypeOf(""))
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestNewFromYamlFile_ErrorOnMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "missing.yaml")

	val := validator.New()
	_, err := NewFromYamlFile(path, false, val)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestNewFromYamlFile_InvalidYaml(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	err := os.WriteFile(path, []byte("key: [invalid yaml"), 0o644)
	require.NoError(t, err)

	val := validator.New()
	_, err = NewFromYamlFile(path, false, val)
	require.Error(t, err)
}

func TestNewFromYamlFile_CustomOptions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	data := map[string]any{"key": "val"}
	yamlBytes, err := yaml.Marshal(data)
	require.NoError(t, err)
	err = os.WriteFile(path, yamlBytes, 0o644)
	require.NoError(t, err)

	val := validator.New()
	r, err := NewFromYamlFile(path, false, val, WithPriority(50), WithKey("custom"))
	require.NoError(t, err)
	assert.Equal(t, 50, r.Priority())
	assert.Equal(t, "custom", r.Key())
}
