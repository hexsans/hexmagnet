package servercfg

import (
	"path/filepath"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()

	assert.Empty(t, cfg.IP)
	assert.Equal(t, 3333, cfg.Port)
	assert.Equal(t, "info", cfg.Log.ConsoleLevel)
	assert.Equal(t, "off", cfg.Log.FileOutputLevel)
	assert.Equal(t, 5, cfg.Log.FileRotator.MaxBackups)
	assert.Equal(t, "text", cfg.Log.FileRotator.Format)
	assert.NotEmpty(t, cfg.Log.FileRotator.Path)
	assert.Empty(t, cfg.EmbedTrackers)
	assert.Equal(t, filepath.Join(".", "data", "torrents"), cfg.TorrentFilePath)
}

func TestValidateTorrentFilePath_Empty(t *testing.T) {
	t.Parallel()

	val := validator.New()
	val.RegisterStructValidation(ValidateTorrentFilePath, Config{})

	cfg := NewDefaultConfig()
	cfg.TorrentFilePath = ""

	err := val.Struct(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestValidateTorrentFilePath_Set(t *testing.T) {
	t.Parallel()

	val := validator.New()
	val.RegisterStructValidation(ValidateTorrentFilePath, Config{})

	cfg := NewDefaultConfig()
	cfg.TorrentFilePath = "/app/torrents"

	err := val.Struct(cfg)
	require.NoError(t, err)
}

func TestNewDefaultConfig_Immutable(t *testing.T) {
	t.Parallel()

	cfg1 := NewDefaultConfig()
	cfg2 := NewDefaultConfig()

	cfg1.Port = 8080
	assert.Equal(t, 8080, cfg1.Port)
	assert.Equal(t, 3333, cfg2.Port)
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	opt := NewModule()
	assert.NotNil(t, opt)
}
