package resolvers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/configcheck"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"github.com/hexsans/hexmagnet/internal/torznab"
	"github.com/hexsans/hexmagnet/internal/webhook"
)

// ConfigValidator runs structural checks and connectivity probes against a
// merged config before it is persisted. Probe functions are fields so tests
// can stub out network-dependent checks.
type ConfigValidator struct {
	Postgres      func(ctx context.Context, cfg postgres.Config) error
	Elasticsearch func(ctx context.Context, cfg elasticsearch.Config) error
	KafkaBrokers  func(ctx context.Context, brokers []string) error
	TMDB          func(ctx context.Context, cfg tmdb.Config) error
	HTTPEndpoint  func(ctx context.Context, endpoint string) error
}

func newConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		Postgres:      configcheck.Postgres,
		Elasticsearch: configcheck.Elasticsearch,
		KafkaBrokers:  configcheck.KafkaBrokers,
		TMDB:          configcheck.TMDB,
		HTTPEndpoint:  configcheck.HTTPEndpoint,
	}
}

// Validate runs all checks for the sections present in the input and returns
// every failure that was found. The merged config data is only used to build
// local structs; no resolver state is touched.
func (v *ConfigValidator) Validate(ctx context.Context, input gen.ConfigInput, data map[string]any) error {
	var errs []error

	if in, ok := input.Dht.ValueOK(); ok && in != nil {
		errs = append(errs, v.checkDHT(ctx, *in, data)...)
	}

	if in, ok := input.Server.ValueOK(); ok && in != nil {
		errs = append(errs, v.checkServer(ctx, *in, data)...)
	}

	if in, ok := input.Classifier.ValueOK(); ok && in != nil {
		errs = append(errs, v.checkClassifier(ctx, *in, data)...)
	}

	if in, ok := input.Storage.ValueOK(); ok && in != nil {
		errs = append(errs, v.checkStorage(ctx, *in, data)...)
	}

	if in, ok := input.Torznab.ValueOK(); ok && in != nil {
		errs = append(errs, v.checkTorznab(*in, data)...)
	}

	if in, ok := input.Webhooks.ValueOK(); ok && in != nil {
		errs = append(errs, v.checkWebhooks(*in, data)...)
	}

	return errors.Join(errs...)
}

func (*ConfigValidator) checkDHT(_ context.Context, input gen.DHTConfigInput, _ map[string]any) []error {
	if p, ok := input.Port.ValueOK(); ok && p != nil && (*p < 1 || *p > 65535) {
		return []error{fmt.Errorf("dht: port must be between 1 and 65535, got %d", *p)}
	}

	return nil
}

func (*ConfigValidator) checkServer(_ context.Context, input gen.ServerConfigInput, data map[string]any) []error {
	rawMap, ok := data["server"].(map[string]any)
	if !ok {
		return nil
	}

	var cfg servercfg.Config
	if err := reloadConfigSection(rawMap, &cfg); err != nil {
		return []error{fmt.Errorf("server: %w", err)}
	}

	var errs []error

	if p, ok := input.Port.ValueOK(); ok && p != nil && (*p < 1 || *p > 65535) {
		errs = append(errs, fmt.Errorf("server: port must be between 1 and 65535, got %d", *p))
	}

	if lg, ok := input.Log.ValueOK(); ok && lg != nil {
		if c, ok := lg.ConsoleLevel.ValueOK(); ok && c != nil && !slices.Contains(logLevels, *c) {
			errs = append(errs, fmt.Errorf("server: log console_level must be one of %v, got %q", logLevels, *c))
		}

		if c, ok := lg.FileOutputLevel.ValueOK(); ok && c != nil && !slices.Contains(logFileLevels, *c) {
			errs = append(errs, fmt.Errorf("server: log file_output_level must be one of %v, got %q", logFileLevels, *c))
		}

		if fr, ok := lg.FileRotator.ValueOK(); ok && fr != nil {
			if f, ok := fr.Format.ValueOK(); ok && f != nil && !slices.Contains([]string{"text", "json"}, *f) {
				errs = append(errs, fmt.Errorf("server: log format must be one of text, json, got %q", *f))
			}
		}

		if err := configcheck.LogPath(cfg.Log); err != nil {
			errs = append(errs, fmt.Errorf("server: log: %w", err))
		}
	}

	if _, ok := input.TorrentFilePath.ValueOK(); ok {
		if err := configcheck.TorrentDir(cfg.TorrentFilePath); err != nil {
			errs = append(errs, fmt.Errorf("server: torrent_file_path: %w", err))
		}
	}

	return errs
}

func (v *ConfigValidator) checkClassifier(ctx context.Context, input gen.ClassifierConfigInput, data map[string]any) []error {
	rawMap, ok := data["classifier"].(map[string]any)
	if !ok {
		return nil
	}

	var cfg classifier.Config
	if err := reloadConfigSection(rawMap, &cfg); err != nil {
		return []error{fmt.Errorf("classifier: %w", err)}
	}

	var errs []error

	if c, ok := input.Concurrency.ValueOK(); ok && c != nil && *c < 1 {
		errs = append(errs, fmt.Errorf("classifier: concurrency must be at least 1, got %d", *c))
	}

	if tf, ok := input.TorrentFilter.ValueOK(); ok && tf != nil {
		if m, ok := tf.Mode.ValueOK(); ok && m != nil && !slices.Contains([]string{"off", "process", "discard"}, *m) {
			errs = append(errs, fmt.Errorf("classifier: torrent_filter mode must be one of off, process, discard, got %q", *m))
		}
	}

	if _, ok := input.Tmdb.ValueOK(); ok {
		if err := v.TMDB(ctx, cfg.Tmdb); err != nil {
			errs = append(errs, fmt.Errorf("classifier: %w", err))
		}
	}

	if llm, ok := input.Llm.ValueOK(); ok && llm != nil {
		if r, ok := llm.ReasoningEffort.ValueOK(); ok && r != nil && *r != "" &&
			!slices.Contains([]string{"none", "low", "medium", "high"}, *r) {
			errs = append(
				errs,
				fmt.Errorf("classifier: llm reasoning_effort must be one of none, low, medium, high, got %q", *r),
			)
		}

		if p, ok := llm.Prompt.ValueOK(); ok && p != nil && len([]rune(*p)) > llmPromptMaxRunes {
			errs = append(
				errs,
				fmt.Errorf("classifier: llm prompt must be at most %d characters, got %d", llmPromptMaxRunes, len([]rune(*p))),
			)
		}

		if cfg.LLM.Enabled && cfg.LLM.Endpoint != "" {
			if err := v.HTTPEndpoint(ctx, cfg.LLM.Endpoint); err != nil {
				errs = append(errs, fmt.Errorf("classifier: llm: %w", err))
			}
		}
	}

	return errs
}

func (v *ConfigValidator) checkStorage(ctx context.Context, input gen.StorageConfigInput, data map[string]any) []error {
	rawMap, ok := data["storage"].(map[string]any)
	if !ok {
		return nil
	}

	var errs []error

	if pgInput, ok := input.Postgres.ValueOK(); ok && pgInput != nil {
		var cfg postgres.Config
		if raw, ok := rawMap["postgres"]; ok {
			if err := reloadConfigSection(raw, &cfg); err != nil {
				errs = append(errs, fmt.Errorf("storage: postgres: %w", err))
			}
		}

		if ssl, ok := pgInput.SslMode.ValueOK(); ok && ssl != nil && !slices.Contains(postgresSSLModes, *ssl) {
			errs = append(errs, fmt.Errorf("storage: postgres ssl_mode must be one of %v, got %q", postgresSSLModes, *ssl))
		}

		if mc, ok := pgInput.MaxConnections.ValueOK(); ok && mc != nil && *mc < 1 {
			errs = append(errs, fmt.Errorf("storage: postgres max_connections must be at least 1, got %d", *mc))
		}

		if err := v.Postgres(ctx, cfg); err != nil {
			errs = append(errs, fmt.Errorf("storage: %w", err))
		}
	}

	if qInput, ok := input.Queue.ValueOK(); ok && qInput != nil {
		var cfg queue.Config
		if raw, ok := rawMap["queue"]; ok {
			if err := reloadConfigSection(raw, &cfg); err != nil {
				errs = append(errs, fmt.Errorf("storage: queue: %w", err))
			}
		}

		if b, ok := qInput.Backend.ValueOK(); ok && b != nil && !slices.Contains([]string{"memory", "kafka"}, *b) {
			errs = append(errs, fmt.Errorf("storage: queue backend must be one of memory, kafka, got %q", *b))
		}

		if cfg.Backend == "kafka" {
			if err := v.KafkaBrokers(ctx, cfg.Kafka.Brokers); err != nil {
				errs = append(errs, fmt.Errorf("storage: %w", err))
			}
		}
	}

	if sInput, ok := input.Search.ValueOK(); ok && sInput != nil {
		var cfg indexer.SearchConfig
		if raw, ok := rawMap["search"]; ok {
			if err := reloadConfigSection(raw, &cfg); err != nil {
				errs = append(errs, fmt.Errorf("storage: search: %w", err))
			}
		}

		if b, ok := sInput.Backend.ValueOK(); ok && b != nil && !slices.Contains([]string{"postgresql", "elasticsearch"}, *b) {
			errs = append(errs, fmt.Errorf("storage: search backend must be one of postgresql, elasticsearch, got %q", *b))
		}

		if cfg.Backend == "elasticsearch" {
			if err := v.Elasticsearch(ctx, cfg.Elasticsearch); err != nil {
				errs = append(errs, fmt.Errorf("storage: %w", err))
			}

			if cfg.Elasticsearch.Embedding.Endpoint != "" {
				if err := v.HTTPEndpoint(ctx, cfg.Elasticsearch.Embedding.Endpoint); err != nil {
					errs = append(errs, fmt.Errorf("storage: search embedding: %w", err))
				}
			}
		}
	}

	return errs
}

func (*ConfigValidator) checkTorznab(input gen.TorznabConfigInput, data map[string]any) []error {
	var errs []error

	if p, ok := input.Path.ValueOK(); ok && p != nil && *p == "" {
		errs = append(errs, fmt.Errorf("torznab: path must not be empty"))
	}

	if m, ok := input.MaxResults.ValueOK(); ok && m != nil && *m < 1 {
		errs = append(errs, fmt.Errorf("torznab: max_results must be at least 1, got %d", *m))
	}

	if cats, ok := input.Categories.ValueOK(); ok {
		if len(cats) == 0 {
			errs = append(errs, fmt.Errorf("torznab: categories must not be empty (use \"*\" for all)"))
		} else {
			for _, c := range cats {
				if c == "*" {
					continue
				}

				cat, err := strconv.Atoi(c)
				if err != nil || cat < 1 {
					errs = append(errs, fmt.Errorf("torznab: invalid category %q (expected a Newznab category ID or \"*\")", c))
				}
			}
		}
	}

	if rawMap, ok := data["torznab"].(map[string]any); ok {
		var cfg torznab.Config
		if err := reloadConfigSection(rawMap, &cfg); err != nil {
			errs = append(errs, fmt.Errorf("torznab: %w", err))
		} else if cfg.Path == "" {
			errs = append(errs, fmt.Errorf("torznab: path must not be empty"))
		}
	}

	return errs
}

func (*ConfigValidator) checkWebhooks(_ gen.WebhooksConfigInput, data map[string]any) []error {
	rawMap, ok := data["webhooks"].(map[string]any)
	if !ok {
		return nil
	}

	var cfg webhook.Config
	if err := reloadConfigSection(rawMap, &cfg); err != nil {
		return []error{fmt.Errorf("webhooks: %w", err)}
	}

	if err := cfg.Validate(); err != nil {
		return []error{err}
	}

	return nil
}

var (
	logLevels        = []string{"debug", "info", "warn", "error"}
	logFileLevels    = []string{"debug", "info", "warn", "error", "off"}
	postgresSSLModes = []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}
)

const llmPromptMaxRunes = 20000
