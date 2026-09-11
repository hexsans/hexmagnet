package resolvers

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/torznab"
	"github.com/hexsans/hexmagnet/internal/webhook"
	"gopkg.in/yaml.v3"
)

// rejectDimsChangeWhileBusy blocks changes to the embedding dimensions while a
// maintenance job (reindex/reclassify) is running. Such a change would trigger
// an automatic reindex, desyncing the running job, the ES index mapping and the
// persisted config.
func (r *mutationResolver) rejectDimsChangeWhileBusy(input gen.ConfigInput) error {
	if r.JobControl == nil || !r.JobControl.Paused() {
		return nil
	}

	storage, ok := input.Storage.ValueOK()
	if !ok || storage == nil {
		return nil
	}

	search, ok := storage.Search.ValueOK()
	if !ok || search == nil {
		return nil
	}

	es, ok := search.Elasticsearch.ValueOK()
	if !ok || es == nil {
		return nil
	}

	embeddingCfg, ok := es.Embedding.ValueOK()
	if !ok || embeddingCfg == nil {
		return nil
	}

	dims, ok := embeddingCfg.Dimensions.ValueOK()
	if !ok || dims == nil {
		return nil
	}

	if *dims != r.SearchCfg.Elasticsearch.Embedding.Dimensions {
		return fmt.Errorf("cannot change embedding dimensions while %s is running", r.JobControl.Reason())
	}

	return nil
}

func readConfigFile(path string) (map[string]any, error) {
	data := make(map[string]any)

	existing, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(existing, &data); err != nil {
			return nil, err
		}
	}

	return data, nil
}

func writeConfigFile(path string, data map[string]any) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0o644)
}

func configToDHT(c dht.Config) gen.DHTConfig {
	return gen.DHTConfig{
		Port:                         uint64(c.Port),
		Responder:                    configToDHTResponder(c.Responder),
		BootstrapNodes:               c.BootstrapNodes,
		ReseedBootstrapNodesInterval: uint64(c.ReseedBootstrapNodesInterval.Seconds()),
	}
}

func configToDHTResponder(c dht.ResponderConfig) gen.DHTResponderConfig {
	return gen.DHTResponderConfig{
		Enabled:         c.Enabled,
		GlobalRateLimit: uint64(c.GlobalRateLimit),
		PerIPRateLimit:  uint64(c.PerIPRateLimit),
	}
}

func configToServer(c servercfg.Config) gen.ServerConfig {
	return gen.ServerConfig{
		IP:   c.IP,
		Port: uint64(c.Port),
		Log: gen.ServerLogConfig{
			ConsoleLevel:    c.Log.ConsoleLevel,
			FileOutputLevel: c.Log.FileOutputLevel,
			FileRotator: gen.ServerFileRotatorConfig{
				Path:       c.Log.FileRotator.Path,
				MaxBackups: uint64(c.Log.FileRotator.MaxBackups),
				Format:     c.Log.FileRotator.Format,
			},
		},
		EmbedTrackers:   c.EmbedTrackers,
		TorrentFilePath: c.TorrentFilePath,
	}
}

func configToClassifier(c classifier.Config) gen.ClassifierConfig {
	return gen.ClassifierConfig{
		Concurrency: uint64(c.Concurrency),
		Llm: gen.LLMConfig{
			Endpoint:        c.LLM.Endpoint,
			APIKey:          maskSecret(c.LLM.APIKey),
			Model:           c.LLM.Model,
			Timeout:         c.LLM.Timeout,
			MaxRetries:      c.LLM.MaxRetries,
			Temperature:     c.LLM.Temperature,
			ReasoningEffort: c.LLM.ReasoningEffort,
			MaxFiles:        c.LLM.MaxFiles,
			Enabled:         c.LLM.Enabled,
		},
		TorrentFilter: gen.TorrentFilterConfig{
			Mode:             string(c.TorrentFilter.Mode),
			TitlePatterns:    c.TorrentFilter.TitlePatterns,
			FilenamePatterns: c.TorrentFilter.FilenamePatterns,
		},
		Tmdb: gen.TMDBConfig{
			Enabled:     c.Tmdb.Enabled,
			AccessToken: maskSecret(c.Tmdb.AccessToken),
			RateLimit:   uint64(c.Tmdb.RateLimit),
		},
	}
}

func configToDHTRequester(c metainforequester.Config) gen.DHTRequesterConfig {
	return gen.DHTRequesterConfig{
		RequestLimit:      uint64(c.RequestLimit),
		RescrapeThreshold: uint64(c.RescrapeThreshold),
		HashDiscoverLimit: uint64(c.HashDiscoverLimit),
	}
}

func configToStorage(pg postgres.Config, search indexer.SearchConfig, q queue.Config) gen.StorageConfig {
	return gen.StorageConfig{
		Postgres: configToPostgres(pg),
		Search:   configToSearch(search),
		Queue:    configToQueue(q),
	}
}

func configToTorznab(c torznab.Config) gen.TorznabConfig {
	return gen.TorznabConfig{
		Enabled:           c.Enabled,
		APIKey:            maskSecret(c.APIKey),
		Path:              c.Path,
		MaxResults:        uint64(c.MaxResults),
		Categories:        c.Categories,
		TrustProxyHeaders: c.TrustProxyHeaders,
	}
}

func configToWebhooks(c webhook.Config) gen.WebhooksConfig {
	headers := make([]gen.WebhookHeader, 0, len(c.Headers))

	for _, k := range slices.Sorted(maps.Keys(c.Headers)) {
		headers = append(headers, gen.WebhookHeader{
			Key:   k,
			Value: maskSecret(c.Headers[k]),
		})
	}

	return gen.WebhooksConfig{
		Enabled:          c.Enabled,
		Urls:             c.Urls,
		Events:           c.Events,
		Categories:       c.Categories,
		TitlePatterns:    c.TitlePatterns,
		FilenamePatterns: c.FilenamePatterns,
		Timeout:          uint64(c.Timeout.Seconds()),
		MaxRetries:       uint64(c.MaxRetries),
		BaseURL:          c.BaseURL,
		Headers:          headers,
		QueueSize:        uint64(c.QueueSize),
	}
}

func configToPostgres(pg postgres.Config) gen.PostgresConfig {
	return gen.PostgresConfig{
		Host:              pg.Host,
		Username:          pg.Username,
		Port:              uint64(pg.Port),
		Database:          pg.Database,
		Password:          maskSecret(pg.Password),
		SslMode:           pg.SSLMode,
		ConnectionTimeout: uint64(pg.ConnectionTimeout),
		SslCertPath:       pg.SSLCertPath,
		SslKeyPath:        pg.SSLKeyPath,
		SslRootCertPath:   pg.SSLRootCertPath,
		MaxConnections:    uint64(pg.MaxConnections),
	}
}

func configToSearch(search indexer.SearchConfig) gen.SearchConfig {
	return gen.SearchConfig{
		Backend: search.Backend,
		Elasticsearch: gen.ElasticsearchConfig{
			Addresses: search.Elasticsearch.Addresses,
			Embedding: gen.EmbeddingConfig{
				Endpoint:           search.Elasticsearch.Embedding.Endpoint,
				Apikey:             maskSecret(search.Elasticsearch.Embedding.APIKey),
				Model:              search.Elasticsearch.Embedding.Model,
				Dimensions:         search.Elasticsearch.Embedding.Dimensions,
				InstructionEnabled: search.Elasticsearch.Embedding.InstructionEnabled,
			},
		},
	}
}

func configToQueue(q queue.Config) gen.QueueConfig {
	return gen.QueueConfig{
		Backend: q.Backend,
		Kafka: gen.KafkaConfig{
			Brokers: q.Kafka.Brokers,
		},
	}
}

func maskSecret(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}

	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}

func getOrCreateSection(data map[string]any, key string) map[string]any {
	section, _ := data[key].(map[string]any)
	if section == nil {
		section = make(map[string]any)
		data[key] = section
	}

	return section
}

func applyDHTInput(data map[string]any, input gen.DHTConfigInput) error {
	dhtSection := getOrCreateSection(data, "dht")
	if v, ok := input.Port.ValueOK(); ok && v != nil {
		dhtSection["port"] = *v
	}

	if v, ok := input.Responder.ValueOK(); ok && v != nil {
		responderSection, _ := dhtSection["responder"].(map[string]any)
		if responderSection == nil {
			responderSection = make(map[string]any)
		}

		if vv, ok := v.Enabled.ValueOK(); ok && vv != nil {
			responderSection["enabled"] = *vv
		}

		if vv, ok := v.GlobalRateLimit.ValueOK(); ok && vv != nil {
			responderSection["global_rate_limit"] = *vv
		}

		if vv, ok := v.PerIPRateLimit.ValueOK(); ok && vv != nil {
			responderSection["per_ip_rate_limit"] = *vv
		}

		dhtSection["responder"] = responderSection
	}

	if v, ok := input.BootstrapNodes.ValueOK(); ok {
		dhtSection["bootstrap_nodes"] = v
	}

	if v, ok := input.ReseedBootstrapNodesInterval.ValueOK(); ok && v != nil {
		dhtSection["reseed_bootstrap_nodes_interval"] = fmt.Sprintf("%ds", *v)
	}

	if v, ok := input.Requester.ValueOK(); ok && v != nil {
		if err := applyDHTRequesterInput(dhtSection, *v); err != nil {
			return err
		}
	}

	data["dht"] = dhtSection

	return nil
}

func applyServerInput(data map[string]any, input gen.ServerConfigInput) error {
	serverSection := getOrCreateSection(data, "server")
	if v, ok := input.IP.ValueOK(); ok && v != nil {
		serverSection["ip"] = *v
	}

	if v, ok := input.Port.ValueOK(); ok && v != nil {
		serverSection["port"] = *v
	}

	if v, ok := input.Log.ValueOK(); ok && v != nil {
		logSection, _ := serverSection["log"].(map[string]any)
		if logSection == nil {
			logSection = make(map[string]any)
		}

		if vv, ok := v.ConsoleLevel.ValueOK(); ok && vv != nil {
			logSection["console_level"] = *vv
		}

		if vv, ok := v.FileOutputLevel.ValueOK(); ok && vv != nil {
			logSection["file_output_level"] = *vv
		}

		if vv, ok := v.FileRotator.ValueOK(); ok && vv != nil {
			rotatorSection, _ := logSection["file_rotator"].(map[string]any)
			if rotatorSection == nil {
				rotatorSection = make(map[string]any)
			}

			if vvv, ok := vv.Path.ValueOK(); ok && vvv != nil {
				rotatorSection["path"] = *vvv
			}

			if vvv, ok := vv.MaxBackups.ValueOK(); ok && vvv != nil {
				rotatorSection["max_backups"] = *vvv
			}

			if vvv, ok := vv.Format.ValueOK(); ok && vvv != nil {
				rotatorSection["format"] = *vvv
			}

			logSection["file_rotator"] = rotatorSection
		}

		serverSection["log"] = logSection
	}

	if v, ok := input.EmbedTrackers.ValueOK(); ok {
		serverSection["embed_trackers"] = v
	}

	if v, ok := input.TorrentFilePath.ValueOK(); ok && v != nil {
		serverSection["torrent_file_path"] = *v
	}

	data["server"] = serverSection

	return nil
}

func applyClassifierInput(data map[string]any, input gen.ClassifierConfigInput) error {
	classifierSection := getOrCreateSection(data, "classifier")
	if v, ok := input.Concurrency.ValueOK(); ok && v != nil {
		classifierSection["concurrency"] = *v
	}

	if v, ok := input.Llm.ValueOK(); ok && v != nil {
		llmSection, _ := classifierSection["llm"].(map[string]any)
		if llmSection == nil {
			llmSection = make(map[string]any)
		}

		if vv, ok := v.Endpoint.ValueOK(); ok && vv != nil {
			llmSection["endpoint"] = *vv
		}

		if vv, ok := v.APIKey.ValueOK(); ok && vv != nil {
			llmSection["api_key"] = *vv
		}

		if vv, ok := v.Model.ValueOK(); ok && vv != nil {
			llmSection["model"] = *vv
		}

		if vv, ok := v.Timeout.ValueOK(); ok && vv != nil {
			llmSection["timeout"] = *vv
		}

		if vv, ok := v.MaxRetries.ValueOK(); ok && vv != nil {
			llmSection["max_retries"] = *vv
		}

		if vv, ok := v.Temperature.ValueOK(); ok && vv != nil {
			llmSection["temperature"] = *vv
		}

		if vv, ok := v.ReasoningEffort.ValueOK(); ok && vv != nil {
			llmSection["reasoning_effort"] = *vv
		}

		if vv, ok := v.MaxFiles.ValueOK(); ok && vv != nil {
			llmSection["max_files"] = *vv
		}

		if vv, ok := v.Enabled.ValueOK(); ok && vv != nil {
			llmSection["enabled"] = *vv
		}

		classifierSection["llm"] = llmSection
	}

	if v, ok := input.TorrentFilter.ValueOK(); ok && v != nil {
		tfSection, _ := classifierSection["torrent_filter"].(map[string]any)
		if tfSection == nil {
			tfSection = make(map[string]any)
		}

		if vv, ok := v.Mode.ValueOK(); ok && vv != nil {
			tfSection["mode"] = *vv
		}

		if vv, ok := v.TitlePatterns.ValueOK(); ok {
			tfSection["title_patterns"] = vv
		}

		if vv, ok := v.FilenamePatterns.ValueOK(); ok {
			tfSection["filename_patterns"] = vv
		}

		classifierSection["torrent_filter"] = tfSection
	}

	if v, ok := input.Tmdb.ValueOK(); ok && v != nil {
		tmdbSection, _ := classifierSection["tmdb"].(map[string]any)
		if tmdbSection == nil {
			tmdbSection = make(map[string]any)
		}

		if vv, ok := v.Enabled.ValueOK(); ok && vv != nil {
			tmdbSection["enabled"] = *vv
		}

		if vv, ok := v.AccessToken.ValueOK(); ok && vv != nil {
			tmdbSection["access_token"] = *vv
		}

		if vv, ok := v.RateLimit.ValueOK(); ok && vv != nil {
			tmdbSection["rate_limit"] = *vv
		}

		classifierSection["tmdb"] = tmdbSection
	}

	data["classifier"] = classifierSection

	return nil
}

func reloadConfigSection(raw any, target any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: target,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
		MatchName: func(mapKey, fieldName string) bool {
			return strings.EqualFold(
				strings.ReplaceAll(mapKey, "_", ""),
				strings.ReplaceAll(fieldName, "_", ""),
			)
		},
	})
	if err != nil {
		return err
	}

	return decoder.Decode(raw)
}

func applyDHTRequesterInput(section map[string]any, input gen.DHTRequesterInput) error {
	sub, _ := section["requester"].(map[string]any)
	if sub == nil {
		sub = make(map[string]any)
	}

	if v, ok := input.RequestLimit.ValueOK(); ok && v != nil {
		sub["request_limit"] = *v
	}

	if v, ok := input.RescrapeThreshold.ValueOK(); ok && v != nil {
		sub["rescrape_threshold"] = *v
	}

	if v, ok := input.HashDiscoverLimit.ValueOK(); ok && v != nil {
		sub["hash_discover_limit"] = *v
	}

	section["requester"] = sub

	return nil
}

func applyStorageInput(data map[string]any, input gen.StorageConfigInput) error {
	storageSection, _ := data["storage"].(map[string]any)
	if storageSection == nil {
		storageSection = make(map[string]any)
	}

	if v, ok := input.Postgres.ValueOK(); ok && v != nil {
		pgSection, _ := storageSection["postgres"].(map[string]any)
		if pgSection == nil {
			pgSection = make(map[string]any)
		}

		if vv, ok := v.Host.ValueOK(); ok && vv != nil {
			pgSection["host"] = *vv
		}

		if vv, ok := v.Username.ValueOK(); ok && vv != nil {
			pgSection["username"] = *vv
		}

		if vv, ok := v.Port.ValueOK(); ok && vv != nil {
			pgSection["port"] = *vv
		}

		if vv, ok := v.Database.ValueOK(); ok && vv != nil {
			pgSection["database"] = *vv
		}

		if vv, ok := v.Password.ValueOK(); ok && vv != nil {
			pgSection["password"] = *vv
		}

		if vv, ok := v.SslMode.ValueOK(); ok && vv != nil {
			pgSection["ssl_mode"] = *vv
		}

		if vv, ok := v.ConnectionTimeout.ValueOK(); ok && vv != nil {
			pgSection["connection_timeout"] = *vv
		}

		if vv, ok := v.SslCertPath.ValueOK(); ok && vv != nil {
			pgSection["ssl_cert_path"] = *vv
		}

		if vv, ok := v.SslKeyPath.ValueOK(); ok && vv != nil {
			pgSection["ssl_key_path"] = *vv
		}

		if vv, ok := v.SslRootCertPath.ValueOK(); ok && vv != nil {
			pgSection["ssl_root_cert_path"] = *vv
		}

		if vv, ok := v.MaxConnections.ValueOK(); ok && vv != nil {
			pgSection["max_connections"] = *vv
		}

		storageSection["postgres"] = pgSection
	}

	if v, ok := input.Queue.ValueOK(); ok && v != nil {
		queueSection, _ := storageSection["queue"].(map[string]any)
		if queueSection == nil {
			queueSection = make(map[string]any)
		}

		if vv, ok := v.Backend.ValueOK(); ok && vv != nil {
			queueSection["backend"] = *vv
		}

		if vv, ok := v.Kafka.ValueOK(); ok && vv != nil {
			kafkaSection, _ := queueSection["kafka"].(map[string]any)
			if kafkaSection == nil {
				kafkaSection = make(map[string]any)
			}

			if vvv, ok := vv.Brokers.ValueOK(); ok {
				kafkaSection["brokers"] = vvv
			}

			queueSection["kafka"] = kafkaSection
		}

		storageSection["queue"] = queueSection
	}

	if v, ok := input.Search.ValueOK(); ok && v != nil {
		searchSection, _ := storageSection["search"].(map[string]any)
		if searchSection == nil {
			searchSection = make(map[string]any)
		}

		if vv, ok := v.Backend.ValueOK(); ok && vv != nil {
			searchSection["backend"] = *vv
		}

		if vv, ok := v.Elasticsearch.ValueOK(); ok && vv != nil {
			esSection, _ := searchSection["elasticsearch"].(map[string]any)
			if esSection == nil {
				esSection = make(map[string]any)
			}

			if vvv, ok := vv.Addresses.ValueOK(); ok {
				esSection["addresses"] = vvv
			}

			if vvv, ok := vv.Embedding.ValueOK(); ok && vvv != nil {
				embSection, _ := esSection["embedding"].(map[string]any)
				if embSection == nil {
					embSection = make(map[string]any)
				}

				if vvvv, ok := vvv.Endpoint.ValueOK(); ok && vvvv != nil {
					embSection["endpoint"] = *vvvv
				}

				if vvvv, ok := vvv.Apikey.ValueOK(); ok && vvvv != nil {
					embSection["api_key"] = *vvvv
				}

				if vvvv, ok := vvv.Model.ValueOK(); ok && vvvv != nil {
					embSection["model"] = *vvvv
				}

				if vvvv, ok := vvv.Dimensions.ValueOK(); ok && vvvv != nil {
					embSection["dimensions"] = *vvvv
				}

				if vvvv, ok := vvv.InstructionEnabled.ValueOK(); ok && vvvv != nil {
					embSection["instruction_enabled"] = *vvvv
				}

				esSection["embedding"] = embSection
			}

			searchSection["elasticsearch"] = esSection
		}

		storageSection["search"] = searchSection
	}

	data["storage"] = storageSection

	return nil
}

func applyTorznabInput(data map[string]any, input gen.TorznabConfigInput) error {
	torznabSection := getOrCreateSection(data, "torznab")

	if v, ok := input.Enabled.ValueOK(); ok && v != nil {
		torznabSection["enabled"] = *v
	}

	if v, ok := input.APIKey.ValueOK(); ok && v != nil {
		// Skip masked values echoed back by the UI so the real key is kept.
		if existing, _ := torznabSection["api_key"].(string); *v != maskSecret(existing) {
			torznabSection["api_key"] = *v
		}
	}

	if v, ok := input.Path.ValueOK(); ok && v != nil {
		torznabSection["path"] = *v
	}

	if v, ok := input.MaxResults.ValueOK(); ok && v != nil {
		torznabSection["max_results"] = *v
	}

	if v, ok := input.Categories.ValueOK(); ok {
		torznabSection["categories"] = v
	}

	if v, ok := input.TrustProxyHeaders.ValueOK(); ok && v != nil {
		torznabSection["trust_proxy_headers"] = *v
	}

	data["torznab"] = torznabSection

	return nil
}

func applyWebhooksInput(data map[string]any, input gen.WebhooksConfigInput) error {
	webhooksSection := getOrCreateSection(data, "webhooks")

	if v, ok := input.Enabled.ValueOK(); ok && v != nil {
		webhooksSection["enabled"] = *v
	}

	if v, ok := input.Urls.ValueOK(); ok {
		webhooksSection["urls"] = v
	}

	if v, ok := input.Events.ValueOK(); ok {
		webhooksSection["events"] = v
	}

	if v, ok := input.Categories.ValueOK(); ok {
		webhooksSection["categories"] = v
	}

	if v, ok := input.TitlePatterns.ValueOK(); ok {
		webhooksSection["title_patterns"] = v
	}

	if v, ok := input.FilenamePatterns.ValueOK(); ok {
		webhooksSection["filename_patterns"] = v
	}

	if v, ok := input.Timeout.ValueOK(); ok && v != nil {
		webhooksSection["timeout"] = fmt.Sprintf("%ds", *v)
	}

	if v, ok := input.MaxRetries.ValueOK(); ok && v != nil {
		webhooksSection["max_retries"] = *v
	}

	if v, ok := input.BaseURL.ValueOK(); ok && v != nil {
		webhooksSection["base_url"] = *v
	}

	if v, ok := input.Headers.ValueOK(); ok && v != nil {
		existing, _ := webhooksSection["headers"].(map[string]any)

		// Replace the whole header set: rows omitted by the caller are
		// removed. Masked values echoed back by the UI keep the stored value.
		headers := make(map[string]any, len(v))

		for _, h := range v {
			if strings.TrimSpace(h.Key) == "" {
				continue
			}

			if cur, present := existing[h.Key]; present {
				if s, isStr := cur.(string); isStr && h.Value == maskSecret(s) {
					headers[h.Key] = s
					continue
				}
			}

			headers[h.Key] = h.Value
		}

		webhooksSection["headers"] = headers
	}

	if v, ok := input.QueueSize.ValueOK(); ok && v != nil {
		webhooksSection["queue_size"] = *v
	}

	data["webhooks"] = webhooksSection

	return nil
}
