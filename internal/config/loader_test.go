package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfigPath_Default(t *testing.T) {
	t.Parallel()

	path := resolveConfigPath()
	assert.Equal(t, "./hexmagnet.yaml", path)
}

func TestResolveConfigPath_WithEnv(t *testing.T) {
	t.Setenv("HEXMAGNET_CONFIG_FILE", "/custom/path/config.yaml")

	path := resolveConfigPath()
	assert.Equal(t, "/custom/path/config.yaml", path)
}

func TestLoad_WithDefaults(t *testing.T) {
	type Server struct {
		Port int
		Host string
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "test_config.yaml")
	t.Setenv("HEXMAGNET_CONFIG_FILE", configPath)

	result, err := Load(
		SpecEntry{Key: "server", DefaultValue: Server{Port: 8080, Host: "localhost"}},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Resolved)

	node, ok := result.Resolved.NodeMap["server"]
	require.True(t, ok)

	server := node.Value.(Server)
	assert.Equal(t, 8080, server.Port)
	assert.Equal(t, "localhost", server.Host)

	_, err = os.Stat(configPath)
	require.NoError(t, err, "default config file should have been created")
}

func TestLoad_WithEnvOverride(t *testing.T) {
	type Server struct {
		Port int
		Host string
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "override_config.yaml")
	t.Setenv("HEXMAGNET_CONFIG_FILE", configPath)

	_ = os.Remove(configPath)
	result, err := Load(
		SpecEntry{Key: "server", DefaultValue: Server{Port: 8080, Host: "localhost"}},
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	server := result.Resolved.NodeMap["server"].Value.(Server)
	assert.Equal(t, 8080, server.Port)
	assert.Equal(t, "localhost", server.Host)
}

func TestLoad_MultipleSpecs(t *testing.T) {
	type Server struct {
		Port int
	}

	type Database struct {
		Host string
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "multi_config.yaml")
	t.Setenv("HEXMAGNET_CONFIG_FILE", configPath)

	result, err := Load(
		SpecEntry{Key: "server", DefaultValue: Server{Port: 3000}},
		SpecEntry{Key: "database", DefaultValue: Database{Host: "db.local"}},
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	server := result.Resolved.NodeMap["server"].Value.(Server)
	assert.Equal(t, 3000, server.Port)

	db := result.Resolved.NodeMap["database"].Value.(Database)
	assert.Equal(t, "db.local", db.Host)
}

func TestLoad_WithValidation(t *testing.T) {
	type PortCfg struct {
		Port int `validate:"gte=1,lte=65535"`
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "valid_config.yaml")
	t.Setenv("HEXMAGNET_CONFIG_FILE", configPath)

	_, err := Load(
		SpecEntry{Key: "port_cfg", DefaultValue: PortCfg{Port: 99999}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}
