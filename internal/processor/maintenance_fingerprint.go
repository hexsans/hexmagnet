package processor

import (
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/jobcontrol"
)

// Fingerprint hashes the classifier configuration that affects classification
// results. It is persisted with reclassify progress so a config change can be
// reported to the user when a pending run is restored after a restart.
func Fingerprint(cfg classifier.Config) string {
	return jobcontrol.Fingerprint(struct {
		Enabled         bool
		Endpoint        string
		Model           string
		Temperature     float64
		ReasoningEffort string
		MaxFiles        int
		Prompt          string
		TorrentFilter   classifier.TorrentFilterConfig
	}{
		Enabled:         cfg.LLM.Enabled,
		Endpoint:        cfg.LLM.Endpoint,
		Model:           cfg.LLM.Model,
		Temperature:     cfg.LLM.Temperature,
		ReasoningEffort: cfg.LLM.ReasoningEffort,
		MaxFiles:        cfg.LLM.MaxFiles,
		Prompt:          cfg.LLM.Prompt,
		TorrentFilter:   cfg.TorrentFilter,
	})
}
