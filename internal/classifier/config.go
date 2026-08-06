package classifier

import (
	"github.com/hexsans/hexmagnet/internal/tmdb"
)

type TorrentFilterMode string

const (
	TorrentFilterOff     TorrentFilterMode = "off"
	TorrentFilterProcess TorrentFilterMode = "process"
	TorrentFilterDiscard TorrentFilterMode = "discard"
)

type TorrentFilterConfig struct {
	Mode             TorrentFilterMode `validate:"oneof=off process discard" yaml:"mode"`
	TitlePatterns    []string          `                                     yaml:"title_patterns"`
	FilenamePatterns []string          `                                     yaml:"filename_patterns"`
}

type Config struct {
	Concurrency   int                 `validate:"gte=1" yaml:"concurrency"`
	LLM           LLMConfig           `                 yaml:"llm"`
	TorrentFilter TorrentFilterConfig `                 yaml:"torrent_filter"`
	Tmdb          tmdb.Config         `                 yaml:"tmdb"`
}

func NewDefaultConfig() Config {
	return Config{
		Concurrency: 10,
		LLM:         NewDefaultLLMConfig(),
		TorrentFilter: TorrentFilterConfig{
			Mode:             TorrentFilterOff,
			TitlePatterns:    []string{},
			FilenamePatterns: []string{},
		},
		Tmdb: tmdb.NewDefaultConfig(),
	}
}
