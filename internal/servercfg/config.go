package servercfg

import (
	"path/filepath"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	// IP is the HTTP server listen address.
	//   ""       → all interfaces, dual-stack (IPv4+IPv6)
	//   "0.0.0.0" → IPv4 only
	//   "::"     → IPv6 only
	IP              string    `yaml:"ip"`
	Port            int       `validate:"gte=1,lte=65535" yaml:"port"`
	Log             LogConfig `yaml:"log"`
	EmbedTrackers   []string  `yaml:"embed_trackers"`
	TorrentFilePath string    `yaml:"torrent_file_path"`
}

type LogConfig struct {
	ConsoleLevel    string            `validate:"oneof=debug info warn error"     yaml:"console_level"`
	FileOutputLevel string            `validate:"oneof=debug info warn error off" yaml:"file_output_level"`
	FileRotator     FileRotatorConfig `                                           yaml:"file_rotator"`
}

type FileRotatorConfig struct {
	Path       string `yaml:"path"`
	MaxBackups int    `validate:"gte=0" yaml:"max_backups"`
	MaxSizeMB  int    `validate:"gte=0" yaml:"max_size_mb"`
	Format     string `validate:"oneof=text json" yaml:"format"`
}

func NewDefaultConfig() Config {
	return Config{
		IP:   "",
		Port: 3333,
		Log: LogConfig{
			ConsoleLevel:    "info",
			FileOutputLevel: "off",
			FileRotator: FileRotatorConfig{
				Path:       filepath.Join(".", "logs"),
				MaxBackups: 5,
				MaxSizeMB:  100,
				Format:     "text",
			},
		},
		EmbedTrackers:   []string{},
		TorrentFilePath: filepath.Join(".", "data", "torrents"),
	}
}

// ValidateTorrentFilePath is a StructLevel validator that requires
// TorrentFilePath to be set, mirroring the FileRotator.Path check.
func ValidateTorrentFilePath(sl validator.StructLevel) {
	c := sl.Current().Interface().(Config)
	if c.TorrentFilePath == "" {
		sl.ReportError(c.TorrentFilePath, "TorrentFilePath", "TorrentFilePath", "required", "")
	}
}
