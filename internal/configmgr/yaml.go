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
		"torznab": map[string]any{
			"enabled":             snap.Torznab.Enabled,
			"api_key":             snap.Torznab.APIKey,
			"path":                snap.Torznab.Path,
			"max_results":         snap.Torznab.MaxResults,
			"categories":          snap.Torznab.Categories,
			"trust_proxy_headers": snap.Torznab.TrustProxyHeaders,
		},
		"webhooks": map[string]any{
			"enabled":           snap.Webhooks.Enabled,
			"urls":              snap.Webhooks.Urls,
			"events":            snap.Webhooks.Events,
			"categories":        snap.Webhooks.Categories,
			"title_patterns":    snap.Webhooks.TitlePatterns,
			"filename_patterns": snap.Webhooks.FilenamePatterns,
			"timeout":           snap.Webhooks.Timeout.String(),
			"max_retries":       snap.Webhooks.MaxRetries,
			"base_url":          snap.Webhooks.BaseURL,
			"headers":           snap.Webhooks.Headers,
			"queue_size":        snap.Webhooks.QueueSize,
		},
	}

	b, err := yaml.Marshal(out)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0o644)
}
