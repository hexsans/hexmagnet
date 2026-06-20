package configmgr

import (
	"os"

	"gopkg.in/yaml.v3"
)

func WriteSnapshotToYAML(path string, snap *Snapshot) error {
	out := map[string]any{
		"dht": map[string]any{
			"port":                            snap.DHT.Port,
			"responder":                       snap.DHT.Responder,
			"bootstrap_nodes":                 snap.DHT.BootstrapNodes,
			"reseed_bootstrap_nodes_interval": snap.DHT.ReseedBootstrapNodesInterval.String(),
			"requester": map[string]any{
				"request_limit":       snap.DHTRequester.RequestLimit,
				"rescrape_threshold":  snap.DHTRequester.RescrapeThreshold,
				"hash_discover_limit": snap.DHTRequester.HashDiscoverLimit,
			},
		},
		"server":     snap.Server,
		"classifier": snap.Classifier,
		"storage": map[string]any{
			"postgres": snap.Postgres,
			"search":   snap.Search,
			"queue":    snap.Queue,
		},
	}

	b, err := yaml.Marshal(out)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0o644)
}
