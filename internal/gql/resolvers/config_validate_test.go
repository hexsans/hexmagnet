package resolvers

import (
	"context"
	"os"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConfigValidator_Validate_collectsAllServerErrors(t *testing.T) {
	t.Parallel()

	v := &ConfigValidator{}
	data := map[string]any{
		"server": map[string]any{
			"port": 99999,
			"log": map[string]any{
				"file_output_level": "info",
				"file_rotator":      map[string]any{"path": "/dev/null/xyz"},
			},
			"torrent_file_path": "/dev/null/xyz2",
		},
	}
	input := gen.ConfigInput{
		Server: graphql.OmittableOf[*gen.ServerConfigInput](&gen.ServerConfigInput{
			Port: graphql.OmittableOf[*uint64](uint64Ptr(99999)),
			Log: graphql.OmittableOf[*gen.ServerLogConfigInput](&gen.ServerLogConfigInput{
				FileOutputLevel: graphql.OmittableOf[*string](testutil.StrPtr("info")),
			}),
			TorrentFilePath: graphql.OmittableOf[*string](testutil.StrPtr("/dev/null/xyz2")),
		}),
	}

	err := v.Validate(context.Background(), input, data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
	assert.Contains(t, err.Error(), "server: log")
	assert.Contains(t, err.Error(), "server: torrent_file_path")
}

func TestConfigValidator_Validate_structuralInputChecks(t *testing.T) {
	t.Parallel()

	v := &ConfigValidator{}
	data := map[string]any{
		"server": map[string]any{
			"log": map[string]any{
				"console_level":     "verbose",
				"file_rotator":      map[string]any{"format": "xml"},
				"file_output_level": "off",
			},
		},
	}
	input := gen.ConfigInput{
		Server: graphql.OmittableOf[*gen.ServerConfigInput](&gen.ServerConfigInput{
			Log: graphql.OmittableOf[*gen.ServerLogConfigInput](&gen.ServerLogConfigInput{
				ConsoleLevel: graphql.OmittableOf[*string](testutil.StrPtr("verbose")),
				FileRotator: graphql.OmittableOf[*gen.ServerFileRotatorConfigInput](&gen.ServerFileRotatorConfigInput{
					Format: graphql.OmittableOf[*string](testutil.StrPtr("xml")),
				}),
			}),
		}),
	}

	err := v.Validate(context.Background(), input, data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "console_level")
	assert.Contains(t, err.Error(), "format must be one of text, json")
}

func TestConfigValidator_Validate_probesOnlyChangedSections(t *testing.T) {
	t.Parallel()

	t.Run("postgres and tmdb only", func(t *testing.T) {
		t.Parallel()

		counters := &probeCounters{}
		v := countingValidator(counters)
		data := probeData()

		input := gen.ConfigInput{
			Storage: graphql.OmittableOf[*gen.StorageConfigInput](&gen.StorageConfigInput{
				Postgres: graphql.OmittableOf[*gen.PostgresConfigInput](&gen.PostgresConfigInput{
					Host: graphql.OmittableOf[*string](testutil.StrPtr("h")),
				}),
			}),
			Classifier: graphql.OmittableOf[*gen.ClassifierConfigInput](&gen.ClassifierConfigInput{
				Tmdb: graphql.OmittableOf[*gen.TMDBConfigInput](&gen.TMDBConfigInput{
					AccessToken: graphql.OmittableOf[*string](testutil.StrPtr("tok")),
				}),
			}),
		}
		require.NoError(t, v.Validate(context.Background(), input, data))
		assert.Equal(t, 1, counters.pg)
		assert.Equal(t, 0, counters.es)
		assert.Equal(t, 0, counters.kafka)
		assert.Equal(t, 1, counters.tmdb)
		assert.Equal(t, 0, counters.http)
	})

	t.Run("kafka and elasticsearch backends", func(t *testing.T) {
		t.Parallel()

		counters := &probeCounters{}
		v := countingValidator(counters)
		data := probeData()

		data["storage"].(map[string]any)["queue"].(map[string]any)["backend"] = "kafka"
		data["storage"].(map[string]any)["search"].(map[string]any)["backend"] = "elasticsearch"
		data["classifier"].(map[string]any)["llm"].(map[string]any)["enabled"] = true
		input := gen.ConfigInput{
			Storage: graphql.OmittableOf[*gen.StorageConfigInput](&gen.StorageConfigInput{
				Queue: graphql.OmittableOf[*gen.QueueConfigInput](&gen.QueueConfigInput{
					Backend: graphql.OmittableOf[*string](testutil.StrPtr("kafka")),
				}),
				Search: graphql.OmittableOf[*gen.SearchConfigInput](&gen.SearchConfigInput{
					Backend: graphql.OmittableOf[*string](testutil.StrPtr("elasticsearch")),
				}),
			}),
			Classifier: graphql.OmittableOf[*gen.ClassifierConfigInput](&gen.ClassifierConfigInput{
				Llm: graphql.OmittableOf[*gen.LLMConfigInput](&gen.LLMConfigInput{
					Enabled: graphql.OmittableOf[*bool](boolPtr(true)),
				}),
			}),
		}
		require.NoError(t, v.Validate(context.Background(), input, data))
		assert.Equal(t, 0, counters.pg)
		assert.Equal(t, 1, counters.es)
		assert.Equal(t, 1, counters.kafka)
		assert.Equal(t, 2, counters.http, "embedding and llm endpoints probed")
	})
}

func probeData() map[string]any {
	return map[string]any{
		"storage": map[string]any{
			"postgres": map[string]any{
				"host": "h", "username": "u", "port": 5432, "database": "d",
				"ssl_mode": "disable", "max_connections": 25,
			},
			"queue": map[string]any{
				"backend": "memory",
				"kafka":   map[string]any{"brokers": []any{"b:9092"}},
			},
			"search": map[string]any{
				"backend": "postgresql",
				"elasticsearch": map[string]any{
					"addresses": []any{"http://es:9200"},
					"embedding": map[string]any{"endpoint": "http://emb:11434", "model": "m", "dimensions": 1024},
				},
			},
		},
		"classifier": map[string]any{
			"tmdb": map[string]any{"enabled": true, "access_token": "tok", "rate_limit": 20},
			"llm":  map[string]any{"enabled": false, "endpoint": "http://llm:8000"},
		},
	}
}

type probeCounters struct {
	pg, es, kafka, http, tmdb int
}

func countingValidator(counters *probeCounters) *ConfigValidator {
	return &ConfigValidator{
		Postgres: func(context.Context, postgres.Config) error {
			counters.pg++
			return nil
		},
		Elasticsearch: func(context.Context, elasticsearch.Config) error {
			counters.es++
			return nil
		},
		KafkaBrokers: func(context.Context, []string) error {
			counters.kafka++
			return nil
		},
		TMDB: func(context.Context, tmdb.Config) error {
			counters.tmdb++
			return nil
		},
		HTTPEndpoint: func(context.Context, string) error {
			counters.http++
			return nil
		},
	}
}

func Test_UpdateConfigMutation_validationFailureRejectsWithoutWrite(t *testing.T) {
	t.Parallel()

	baseYAML := map[string]any{
		"server": map[string]any{
			"ip":   "127.0.0.1",
			"port": 4444,
		},
	}

	path := tempYAML(t, baseYAML)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	resolver := fixedResolver()
	resolver.ConfigFilePath = path
	resolver.Logger = zap.NewNop().Sugar()

	dispatched := false
	cm := configmgr.NewManager(&configmgr.Snapshot{}, "", nil, zap.NewNop().Sugar())
	cm.Subscribe(context.Background(), "test", func(context.Context, *configmgr.Snapshot) error {
		dispatched = true
		return nil
	}, configmgr.ApplySync)
	resolver.ConfigManager = cm

	input := gen.ConfigInput{
		Server: graphql.OmittableOf[*gen.ServerConfigInput](&gen.ServerConfigInput{
			Port:            graphql.OmittableOf[*uint64](uint64Ptr(9090)),
			TorrentFilePath: graphql.OmittableOf[*string](testutil.StrPtr("/dev/null/torrents")),
		}),
	}

	m := &mutationResolver{resolver}
	_, err = m.UpdateConfig(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config validation failed")
	assert.Contains(t, err.Error(), "torrent_file_path")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, before, after, "config file must be unchanged on validation failure")

	assert.Equal(t, int64(3333), int64(resolver.ServerCfg.Port), "resolver structs must keep old values")
	assert.False(t, dispatched, "subscribers must not be dispatched on validation failure")
}

func Test_UpdateConfigMutation_validationFailureCollectsAll(t *testing.T) {
	t.Parallel()

	baseYAML := map[string]any{
		"server": map[string]any{
			"ip":   "127.0.0.1",
			"port": 4444,
			"log": map[string]any{
				"file_output_level": "off",
			},
		},
	}

	path := tempYAML(t, baseYAML)
	resolver := fixedResolver()
	resolver.ConfigFilePath = path
	resolver.Logger = zap.NewNop().Sugar()

	input := gen.ConfigInput{
		Server: graphql.OmittableOf[*gen.ServerConfigInput](&gen.ServerConfigInput{
			Port: graphql.OmittableOf[*uint64](uint64Ptr(99999)),
			Log: graphql.OmittableOf[*gen.ServerLogConfigInput](&gen.ServerLogConfigInput{
				FileOutputLevel: graphql.OmittableOf[*string](testutil.StrPtr("info")),
				FileRotator: graphql.OmittableOf[*gen.ServerFileRotatorConfigInput](&gen.ServerFileRotatorConfigInput{
					Path: graphql.OmittableOf[*string](testutil.StrPtr("/dev/null/logs")),
				}),
			}),
			TorrentFilePath: graphql.OmittableOf[*string](testutil.StrPtr("/dev/null/torrents")),
		}),
	}

	m := &mutationResolver{resolver}
	_, err := m.UpdateConfig(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
	assert.Contains(t, err.Error(), "server: log")
	assert.Contains(t, err.Error(), "server: torrent_file_path")
}
