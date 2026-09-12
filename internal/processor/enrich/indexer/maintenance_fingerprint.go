package indexer

import (
	"sort"

	"github.com/hexsans/hexmagnet/internal/jobcontrol"
)

// Fingerprint hashes the search configuration that affects index contents. It
// is persisted with reindex progress so a config change can be reported to the
// user when a pending run is restored after a restart.
func Fingerprint(cfg SearchConfig) string {
	addresses := append([]string(nil), cfg.Elasticsearch.Addresses...)
	sort.Strings(addresses)

	return jobcontrol.Fingerprint(struct {
		Backend            string
		Addresses          []string
		Endpoint           string
		Model              string
		Dimensions         int
		InstructionEnabled bool
	}{
		Backend:            cfg.Backend,
		Addresses:          addresses,
		Endpoint:           cfg.Elasticsearch.Embedding.Endpoint,
		Model:              cfg.Elasticsearch.Embedding.Model,
		Dimensions:         cfg.Elasticsearch.Embedding.Dimensions,
		InstructionEnabled: cfg.Elasticsearch.Embedding.InstructionEnabled,
	})
}
