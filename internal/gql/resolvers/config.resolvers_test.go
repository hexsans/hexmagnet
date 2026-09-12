package resolvers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/elasticsearch"
	"github.com/hexsans/hexmagnet/internal/elasticsearch/embedding"
	"github.com/hexsans/hexmagnet/internal/gql/gqlmodel/gen"
	"github.com/hexsans/hexmagnet/internal/jobcontrol"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/queue/kafka"
	"github.com/hexsans/hexmagnet/internal/search"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/hexsans/hexmagnet/internal/tmdb"
	"github.com/hexsans/hexmagnet/internal/torznab"
	"github.com/hexsans/hexmagnet/internal/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func Test_maskSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "empty", input: "", expect: ""},
		{name: "single char", input: "a", expect: "*"},
		{name: "four chars", input: "abcd", expect: "****"},
		{name: "five chars", input: "abcde", expect: "*bcde"},
		{name: "long token", input: "my-super-secret-api-key-1234", expect: "************************1234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, maskSecret(tt.input))
		})
	}
}

func Test_configToDHT(t *testing.T) {
	t.Parallel()

	cfg := dht.Config{
		Port: 3334,
		Responder: dht.ResponderConfig{
			Enabled:         true,
			GlobalRateLimit: 50,
			PerIPRateLimit:  1,
		},
		BootstrapNodes:               []string{"node1:6881", "node2:6881"},
		ReseedBootstrapNodesInterval: 2 * time.Minute,
	}

	got := configToDHT(cfg)

	assert.Equal(t, uint64(3334), got.Port)
	assert.True(t, got.Responder.Enabled)
	assert.Equal(t, uint64(50), got.Responder.GlobalRateLimit)
	assert.Equal(t, uint64(1), got.Responder.PerIPRateLimit)
	assert.Equal(t, []string{"node1:6881", "node2:6881"}, got.BootstrapNodes)
	assert.Equal(t, uint64(120), got.ReseedBootstrapNodesInterval)
}

func Test_configToServer(t *testing.T) {
	t.Parallel()

	cfg := servercfg.Config{
		IP:   "127.0.0.1",
		Port: 8080,
		Log: servercfg.LogConfig{
			ConsoleLevel:    "debug",
			FileOutputLevel: "info",
			FileRotator: servercfg.FileRotatorConfig{
				Path:       "/var/log/app",
				MaxBackups: 7,
				Format:     "json",
			},
		},
		EmbedTrackers:   []string{"tracker1", "tracker2"},
		TorrentFilePath: "/app/torrents",
	}

	got := configToServer(cfg)

	assert.Equal(t, "127.0.0.1", got.IP)
	assert.Equal(t, uint64(8080), got.Port)
	assert.Equal(t, "debug", got.Log.ConsoleLevel)
	assert.Equal(t, "info", got.Log.FileOutputLevel)
	assert.Equal(t, "/var/log/app", got.Log.FileRotator.Path)
	assert.Equal(t, uint64(7), got.Log.FileRotator.MaxBackups)
	assert.Equal(t, "json", got.Log.FileRotator.Format)
	assert.Equal(t, []string{"tracker1", "tracker2"}, got.EmbedTrackers)
	assert.Equal(t, "/app/torrents", got.TorrentFilePath)
}

func Test_configToClassifier_secretsMasked(t *testing.T) {
	t.Parallel()

	cfg := classifier.Config{
		Concurrency: 5,
		LLM: classifier.LLMConfig{
			Endpoint:        "https://api.llm.local",
			APIKey:          "sk-abcdefghijklmnop",
			Model:           "gpt-4",
			Timeout:         30,
			MaxRetries:      3,
			Temperature:     0.7,
			ReasoningEffort: "high",
			MaxFiles:        10,
			Enabled:         true,
		},
		TorrentFilter: classifier.TorrentFilterConfig{
			Mode:             classifier.TorrentFilterDiscard,
			TitlePatterns:    []string{"*.mp4"},
			FilenamePatterns: []string{"*.pdf"},
		},
		Tmdb: tmdb.Config{
			Enabled:     true,
			AccessToken: "tok_1234567890abcdef",
			RateLimit:   30,
		},
	}

	got := configToClassifier(cfg)

	assert.Equal(t, uint64(5), got.Concurrency)

	assert.Equal(t, "https://api.llm.local", got.Llm.Endpoint)
	assert.Equal(t, "***************mnop", got.Llm.APIKey)
	assert.Equal(t, "gpt-4", got.Llm.Model)
	assert.Equal(t, 30, got.Llm.Timeout)
	assert.Equal(t, 3, got.Llm.MaxRetries)
	assert.InEpsilon(t, 0.7, got.Llm.Temperature, 1e-6)
	assert.Equal(t, "high", got.Llm.ReasoningEffort)
	assert.Equal(t, 10, got.Llm.MaxFiles)
	assert.True(t, got.Llm.Enabled)

	assert.Equal(t, "discard", got.TorrentFilter.Mode)
	assert.Equal(t, []string{"*.mp4"}, got.TorrentFilter.TitlePatterns)
	assert.Equal(t, []string{"*.pdf"}, got.TorrentFilter.FilenamePatterns)

	assert.True(t, got.Tmdb.Enabled)
	assert.Equal(t, "****************cdef", got.Tmdb.AccessToken)
	assert.Equal(t, uint64(30), got.Tmdb.RateLimit)
}

func Test_configToDHTRequester(t *testing.T) {
	t.Parallel()

	cfg := metainforequester.Config{
		RequestLimit:      100,
		RescrapeThreshold: 86400,
		HashDiscoverLimit: 10,
	}

	got := configToDHTRequester(cfg)
	assert.Equal(t, uint64(100), got.RequestLimit)
	assert.Equal(t, uint64(86400), got.RescrapeThreshold)
	assert.Equal(t, uint64(10), got.HashDiscoverLimit)
}

func Test_configToPostgres_secretMasked(t *testing.T) {
	t.Parallel()

	cfg := postgres.Config{
		Host:              "db.example.com",
		Username:          "app_user",
		Port:              5432,
		Database:          "hexmagnet",
		Password:          "super-secret-password-xyz",
		SSLMode:           "verify-full",
		ConnectionTimeout: 30,
		SSLCertPath:       "/certs/cert.pem",
		SSLKeyPath:        "/certs/key.pem",
		SSLRootCertPath:   "/certs/root.pem",
		MaxConnections:    50,
	}

	got := configToPostgres(cfg)

	assert.Equal(t, "db.example.com", got.Host)
	assert.Equal(t, "app_user", got.Username)
	assert.Equal(t, uint64(5432), got.Port)
	assert.Equal(t, "hexmagnet", got.Database)
	assert.Equal(t, "*********************-xyz", got.Password)
	assert.Equal(t, "verify-full", got.SslMode)
	assert.Equal(t, uint64(30), got.ConnectionTimeout)
	assert.Equal(t, "/certs/cert.pem", got.SslCertPath)
	assert.Equal(t, "/certs/key.pem", got.SslKeyPath)
	assert.Equal(t, "/certs/root.pem", got.SslRootCertPath)
	assert.Equal(t, uint64(50), got.MaxConnections)
}

func Test_configToSearch(t *testing.T) {
	t.Parallel()

	cfg := indexer.SearchConfig{
		Backend: "elasticsearch",
		Elasticsearch: elasticsearch.Config{
			Addresses: []string{"http://es:9200", "http://es:9201"},
			Embedding: embedding.Config{
				Endpoint:           "http://ollama:11434/v1",
				APIKey:             "sk-embed-apikey-1234567890abcdef",
				Model:              "bge-m3",
				Dimensions:         1024,
				InstructionEnabled: true,
			},
		},
	}

	got := configToSearch(cfg)
	assert.Equal(t, "elasticsearch", got.Backend)
	assert.Equal(t, []string{"http://es:9200", "http://es:9201"}, got.Elasticsearch.Addresses)
	assert.Equal(t, "http://ollama:11434/v1", got.Elasticsearch.Embedding.Endpoint)
	assert.Equal(t, "****************************cdef", got.Elasticsearch.Embedding.Apikey)
	assert.Equal(t, "bge-m3", got.Elasticsearch.Embedding.Model)
	assert.Equal(t, 1024, got.Elasticsearch.Embedding.Dimensions)
	assert.True(t, got.Elasticsearch.Embedding.InstructionEnabled)
}

func Test_configToQueue(t *testing.T) {
	t.Parallel()

	cfg := queue.Config{
		Backend: "kafka",
		Kafka: kafka.Config{
			Brokers: []string{"kafka1:9092", "kafka2:9092"},
		},
	}

	got := configToQueue(cfg)
	assert.Equal(t, "kafka", got.Backend)
	assert.Equal(t, []string{"kafka1:9092", "kafka2:9092"}, got.Kafka.Brokers)
}

func Test_configToStorage(t *testing.T) {
	t.Parallel()

	pg := postgres.Config{
		Host:           "pg.local",
		Username:       "u",
		Port:           5432,
		Database:       "db",
		Password:       "pw",
		SSLMode:        "disable",
		MaxConnections: 25,
	}
	search := indexer.SearchConfig{Backend: "postgresql", Elasticsearch: elasticsearch.Config{Addresses: []string{"http://es:9200"}}}
	q := queue.Config{Backend: "memory", Kafka: kafka.Config{Brokers: []string{"k:9092"}}}

	got := configToStorage(pg, search, q)
	assert.Equal(t, "pg.local", got.Postgres.Host)
	assert.Equal(t, "postgresql", got.Search.Backend)
	assert.Equal(t, "memory", got.Queue.Backend)
	assert.Equal(t, []string{"http://es:9200"}, got.Search.Elasticsearch.Addresses)
	assert.Equal(t, []string{"k:9092"}, got.Queue.Kafka.Brokers)
}

func uint64Ptr(v uint64) *uint64 { return &v }

func boolPtr(v bool) *bool { return &v }

func strPtr(v string) *string { return &v }

func Test_applyDHTInput(t *testing.T) {
	t.Parallel()

	t.Run("partial update sets only specified fields", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"dht": map[string]any{
				"port": 3333,
			},
		}

		input := gen.DHTConfigInput{
			Port: graphql.OmittableOf[*uint64](uint64Ptr(4444)),
			Requester: graphql.OmittableOf[*gen.DHTRequesterInput](&gen.DHTRequesterInput{
				RequestLimit: graphql.OmittableOf[*uint64](uint64Ptr(100)),
			}),
		}

		err := applyDHTInput(data, input)
		require.NoError(t, err)

		dht := data["dht"].(map[string]any)
		assert.Equal(t, uint64(4444), dht["port"])
		requester := dht["requester"].(map[string]any)
		assert.Equal(t, uint64(100), requester["request_limit"])
	})

	t.Run("creates section if missing", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{}

		input := gen.DHTConfigInput{
			Port: graphql.OmittableOf[*uint64](uint64Ptr(3334)),
		}

		err := applyDHTInput(data, input)
		require.NoError(t, err)

		dht := data["dht"].(map[string]any)
		assert.Equal(t, uint64(3334), dht["port"])
	})

	t.Run("sets nested responder fields", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"dht": map[string]any{
				"port": 3333,
				"responder": map[string]any{
					"enabled":           false,
					"global_rate_limit": 10,
				},
			},
		}

		input := gen.DHTConfigInput{
			Responder: graphql.OmittableOf[*gen.DHTResponderConfigInput](&gen.DHTResponderConfigInput{
				Enabled:        graphql.OmittableOf[*bool](boolPtr(true)),
				PerIPRateLimit: graphql.OmittableOf[*uint64](uint64Ptr(5)),
			}),
		}

		err := applyDHTInput(data, input)
		require.NoError(t, err)

		dht := data["dht"].(map[string]any)
		responder := dht["responder"].(map[string]any)
		assert.Equal(t, true, responder["enabled"])
		assert.Equal(t, 10, responder["global_rate_limit"])
		assert.Equal(t, uint64(5), responder["per_ip_rate_limit"])
	})

	t.Run("sets bootstrap nodes", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"dht": map[string]any{}}

		input := gen.DHTConfigInput{
			BootstrapNodes: graphql.OmittableOf[[]string]([]string{"a:1", "b:2"}),
		}

		err := applyDHTInput(data, input)
		require.NoError(t, err)

		dht := data["dht"].(map[string]any)
		assert.Equal(t, []string{"a:1", "b:2"}, dht["bootstrap_nodes"])
	})
}

func Test_applyServerInput(t *testing.T) {
	t.Parallel()

	t.Run("sets top-level fields", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"server": map[string]any{"ip": "0.0.0.0"}}

		input := gen.ServerConfigInput{
			IP: graphql.OmittableOf[*string](testutil.StrPtr("127.0.0.1")),
		}

		err := applyServerInput(data, input)
		require.NoError(t, err)

		server := data["server"].(map[string]any)
		assert.Equal(t, "127.0.0.1", server["ip"])
	})

	t.Run("sets nested log and file rotator", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"server": map[string]any{}}

		input := gen.ServerConfigInput{
			Log: graphql.OmittableOf[*gen.ServerLogConfigInput](&gen.ServerLogConfigInput{
				ConsoleLevel: graphql.OmittableOf[*string](testutil.StrPtr("warn")),
				FileRotator: graphql.OmittableOf[*gen.ServerFileRotatorConfigInput](&gen.ServerFileRotatorConfigInput{
					MaxBackups: graphql.OmittableOf[*uint64](uint64Ptr(14)),
				}),
			}),
		}

		err := applyServerInput(data, input)
		require.NoError(t, err)

		server := data["server"].(map[string]any)
		log := server["log"].(map[string]any)
		assert.Equal(t, "warn", log["console_level"])
		rotator := log["file_rotator"].(map[string]any)
		assert.Equal(t, uint64(14), rotator["max_backups"])
	})

	t.Run("sets embed trackers", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"server": map[string]any{}}

		input := gen.ServerConfigInput{
			EmbedTrackers: graphql.OmittableOf[[]string]([]string{"t1", "t2"}),
		}

		err := applyServerInput(data, input)
		require.NoError(t, err)

		server := data["server"].(map[string]any)
		assert.Equal(t, []string{"t1", "t2"}, server["embed_trackers"])
	})

	t.Run("sets torrent file path", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"server": map[string]any{}}

		input := gen.ServerConfigInput{
			TorrentFilePath: graphql.OmittableOf[*string](ptr("./custom/torrents")),
		}

		err := applyServerInput(data, input)
		require.NoError(t, err)

		server := data["server"].(map[string]any)
		assert.Equal(t, "./custom/torrents", server["torrent_file_path"])
	})
}

func Test_applyClassifierInput(t *testing.T) {
	t.Parallel()

	t.Run("sets top-level fields", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"classifier": map[string]any{}}

		input := gen.ClassifierConfigInput{
			Concurrency: graphql.OmittableOf[*uint64](uint64Ptr(8)),
		}

		err := applyClassifierInput(data, input)
		require.NoError(t, err)

		cls := data["classifier"].(map[string]any)
		assert.Equal(t, uint64(8), cls["concurrency"])
	})

	t.Run("sets llm subsection", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"classifier": map[string]any{}}

		input := gen.ClassifierConfigInput{
			Llm: graphql.OmittableOf[*gen.LLMConfigInput](&gen.LLMConfigInput{
				Endpoint: graphql.OmittableOf[*string](testutil.StrPtr("https://llm.local")),
				Enabled:  graphql.OmittableOf[*bool](boolPtr(true)),
			}),
		}

		err := applyClassifierInput(data, input)
		require.NoError(t, err)

		cls := data["classifier"].(map[string]any)
		llmSection := cls["llm"].(map[string]any)
		assert.Equal(t, "https://llm.local", llmSection["endpoint"])
		assert.Equal(t, true, llmSection["enabled"])
	})

	t.Run("sets torrent filter", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"classifier": map[string]any{}}

		input := gen.ClassifierConfigInput{
			TorrentFilter: graphql.OmittableOf[*gen.TorrentFilterConfigInput](&gen.TorrentFilterConfigInput{
				Mode:          graphql.OmittableOf[*string](testutil.StrPtr("discard")),
				TitlePatterns: graphql.OmittableOf[[]string]([]string{"*.mkv"}),
			}),
		}

		err := applyClassifierInput(data, input)
		require.NoError(t, err)

		cls := data["classifier"].(map[string]any)
		tf := cls["torrent_filter"].(map[string]any)
		assert.Equal(t, "discard", tf["mode"])
		assert.Equal(t, []string{"*.mkv"}, tf["title_patterns"])
	})

	t.Run("sets tmdb subsection", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"classifier": map[string]any{}}

		input := gen.ClassifierConfigInput{
			Tmdb: graphql.OmittableOf[*gen.TMDBConfigInput](&gen.TMDBConfigInput{
				Enabled:     graphql.OmittableOf[*bool](boolPtr(false)),
				AccessToken: graphql.OmittableOf[*string](testutil.StrPtr("new-token")),
			}),
		}

		err := applyClassifierInput(data, input)
		require.NoError(t, err)

		cls := data["classifier"].(map[string]any)
		tmdbSection := cls["tmdb"].(map[string]any)
		assert.Equal(t, false, tmdbSection["enabled"])
		assert.Equal(t, "new-token", tmdbSection["access_token"])
	})
}

func Test_applyDHTRequesterInput(t *testing.T) {
	t.Parallel()

	data := map[string]any{"dht": map[string]any{}}

	input := gen.DHTRequesterInput{
		RequestLimit: graphql.OmittableOf[*uint64](uint64Ptr(75)),
	}

	dhtSection := data["dht"].(map[string]any)
	err := applyDHTRequesterInput(dhtSection, input)
	require.NoError(t, err)

	section := dhtSection["requester"].(map[string]any)
	assert.Equal(t, uint64(75), section["request_limit"])
}

func Test_applyStorageInput(t *testing.T) {
	t.Parallel()

	t.Run("sets postgres fields", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"storage": map[string]any{}}

		input := gen.StorageConfigInput{
			Postgres: graphql.OmittableOf[*gen.PostgresConfigInput](&gen.PostgresConfigInput{
				Host: graphql.OmittableOf[*string](testutil.StrPtr("pg.new")),
				Port: graphql.OmittableOf[*uint64](uint64Ptr(5433)),
			}),
		}

		err := applyStorageInput(data, input)
		require.NoError(t, err)

		storage := data["storage"].(map[string]any)
		pg := storage["postgres"].(map[string]any)
		assert.Equal(t, "pg.new", pg["host"])
		assert.Equal(t, uint64(5433), pg["port"])
	})

	t.Run("sets queue with kafka", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"storage": map[string]any{}}

		input := gen.StorageConfigInput{
			Queue: graphql.OmittableOf[*gen.QueueConfigInput](&gen.QueueConfigInput{
				Backend: graphql.OmittableOf[*string](testutil.StrPtr("kafka")),
				Kafka: graphql.OmittableOf[*gen.KafkaConfigInput](&gen.KafkaConfigInput{
					Brokers: graphql.OmittableOf[[]string]([]string{"k1:9092", "k2:9092"}),
				}),
			}),
		}

		err := applyStorageInput(data, input)
		require.NoError(t, err)

		storage := data["storage"].(map[string]any)
		queueSection := storage["queue"].(map[string]any)
		assert.Equal(t, "kafka", queueSection["backend"])
		kafkaSection := queueSection["kafka"].(map[string]any)
		assert.Equal(t, []string{"k1:9092", "k2:9092"}, kafkaSection["brokers"])
	})

	t.Run("sets search with elasticsearch", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{"storage": map[string]any{}}

		input := gen.StorageConfigInput{
			Search: graphql.OmittableOf[*gen.SearchConfigInput](&gen.SearchConfigInput{
				Backend: graphql.OmittableOf[*string](testutil.StrPtr("elasticsearch")),
				Elasticsearch: graphql.OmittableOf[*gen.ElasticsearchConfigInput](&gen.ElasticsearchConfigInput{
					Addresses: graphql.OmittableOf[[]string]([]string{"http://es:9200"}),
				}),
			}),
		}

		err := applyStorageInput(data, input)
		require.NoError(t, err)

		storage := data["storage"].(map[string]any)
		search := storage["search"].(map[string]any)
		assert.Equal(t, "elasticsearch", search["backend"])
		es := search["elasticsearch"].(map[string]any)
		assert.Equal(t, []string{"http://es:9200"}, es["addresses"])
	})
}

func Test_readConfigFile(t *testing.T) {
	t.Parallel()

	t.Run("returns empty map for missing file", func(t *testing.T) {
		t.Parallel()

		data, err := readConfigFile("/tmp/nonexistent-file-12345.yaml")
		require.NoError(t, err)
		assert.Empty(t, data)
	})

	t.Run("reads existing yaml", func(t *testing.T) {
		t.Parallel()
		path := tempYAML(t, map[string]any{
			"server": map[string]any{
				"port": 8080,
				"ip":   "0.0.0.0",
			},
		})

		data, err := readConfigFile(path)
		require.NoError(t, err)

		server := data["server"].(map[string]any)
		assert.Equal(t, 8080, server["port"])
		assert.Equal(t, "0.0.0.0", server["ip"])
	})
}

func Test_writeConfigFile(t *testing.T) {
	t.Parallel()

	path := tempYAML(t, nil)

	data := map[string]any{
		"dht": map[string]any{
			"port": 3334,
		},
	}

	err := writeConfigFile(path, data)
	require.NoError(t, err)

	written, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed map[string]any

	err = yaml.Unmarshal(written, &parsed)
	require.NoError(t, err)
	assert.Equal(t, 3334, parsed["dht"].(map[string]any)["port"])
}

func tempYAML(t *testing.T, content map[string]any) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	require.NoError(t, err)

	defer f.Close()

	if content != nil {
		out, err := yaml.Marshal(content)
		require.NoError(t, err)
		_, err = f.Write(out)
		require.NoError(t, err)
	}

	return f.Name()
}

func Test_ConfigQuery(t *testing.T) {
	t.Parallel()

	resolver := fixedResolver()

	q := &queryResolver{resolver}
	cfg, err := q.Config(context.Background())
	require.NoError(t, err)

	assert.Equal(t, uint64(3334), cfg.Dht.Port)
	assert.Equal(t, "127.0.0.1", cfg.Server.IP)
	assert.Equal(t, "pg.local", cfg.Storage.Postgres.Host)
	assert.Equal(t, "postgresql", cfg.Storage.Search.Backend)
	assert.Equal(t, "memory", cfg.Storage.Queue.Backend)
	assert.Equal(t, "***************abcd", cfg.Classifier.Tmdb.AccessToken)
	assert.Equal(t, "****************7890", cfg.Storage.Postgres.Password)
	assert.Equal(t, "**********7890", cfg.Classifier.Llm.APIKey)
	assert.Equal(t, "***********************7890", cfg.Storage.Search.Elasticsearch.Embedding.Apikey)
	assert.Equal(t, []string{"http://es:9200"}, cfg.Storage.Search.Elasticsearch.Addresses)
}

func Test_UpdateConfigMutation_partialUpdate(t *testing.T) {
	t.Parallel()

	baseYAML := map[string]any{
		"server": map[string]any{
			"ip":   "0.0.0.0",
			"port": 3333,
		},
		"dht": map[string]any{
			"port": 3334,
		},
		"classifier": map[string]any{
			"workflow":    "default",
			"concurrency": 10,
		},
	}

	path := tempYAML(t, baseYAML)
	resolver := fixedResolver()
	resolver.ConfigFilePath = path

	input := gen.ConfigInput{
		Server: graphql.OmittableOf[*gen.ServerConfigInput](&gen.ServerConfigInput{
			Port: graphql.OmittableOf[*uint64](uint64Ptr(9090)),
		}),
		Classifier: graphql.OmittableOf[*gen.ClassifierConfigInput](&gen.ClassifierConfigInput{
			Concurrency: graphql.OmittableOf[*uint64](uint64Ptr(20)),
		}),
	}

	m := &mutationResolver{resolver}

	_, err := m.UpdateConfig(context.Background(), input)
	require.NoError(t, err)

	data, err := readConfigFile(path)
	require.NoError(t, err)

	server := data["server"].(map[string]any)
	assert.Equal(t, "0.0.0.0", server["ip"], "unchanged field preserved")
	assert.Equal(t, 9090, server["port"], "updated field changed")

	cls := data["classifier"].(map[string]any)
	assert.Equal(t, 20, cls["concurrency"], "updated field changed")
	assert.Equal(t, "default", cls["workflow"], "unchanged field preserved")

	dht := data["dht"].(map[string]any)
	assert.Equal(t, 3334, dht["port"], "unchanged section preserved")
}

func Test_UpdateConfigMutation_dimsGuard(t *testing.T) {
	t.Parallel()

	resolver := fixedResolver()
	resolver.SearchCfg.Elasticsearch.Embedding.Dimensions = 1024
	resolver.JobControl = jobcontrol.NewController()
	m := &mutationResolver{resolver}

	buildInput := func(dims int) gen.ConfigInput {
		return gen.ConfigInput{
			Storage: graphql.OmittableOf[*gen.StorageConfigInput](&gen.StorageConfigInput{
				Search: graphql.OmittableOf[*gen.SearchConfigInput](&gen.SearchConfigInput{
					Elasticsearch: graphql.OmittableOf[*gen.ElasticsearchConfigInput](&gen.ElasticsearchConfigInput{
						Embedding: graphql.OmittableOf[*gen.EmbeddingConfigInput](&gen.EmbeddingConfigInput{
							Dimensions: graphql.OmittableOf[*int](&dims),
						}),
					}),
				}),
			}),
		}
	}

	require.NoError(t, m.rejectDimsChangeWhileBusy(buildInput(2048)), "idle job must allow dims change")

	require.True(t, resolver.JobControl.TryAcquire(jobcontrol.JobReindex))
	defer resolver.JobControl.Release(jobcontrol.JobReindex)

	err := m.rejectDimsChangeWhileBusy(buildInput(2048))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot change embedding dimensions")

	require.NoError(t, m.rejectDimsChangeWhileBusy(buildInput(1024)), "unchanged dims must be allowed")
}

func Test_UpdateConfigMutation_createsMissingSections(t *testing.T) {
	t.Parallel()

	baseYAML := map[string]any{
		"server": map[string]any{
			"ip": "0.0.0.0",
		},
	}

	path := tempYAML(t, baseYAML)
	resolver := fixedResolver()
	resolver.ConfigFilePath = path

	input := gen.ConfigInput{
		Dht: graphql.OmittableOf[*gen.DHTConfigInput](&gen.DHTConfigInput{
			Port: graphql.OmittableOf[*uint64](uint64Ptr(5555)),
		}),
	}

	m := &mutationResolver{resolver}

	_, err := m.UpdateConfig(context.Background(), input)
	require.NoError(t, err)

	data, err := readConfigFile(path)
	require.NoError(t, err)

	dht := data["dht"].(map[string]any)
	assert.Equal(t, 5555, dht["port"])
}

func Test_UpdateConfigMutation_emptyInputDoesNotChangeFile(t *testing.T) {
	t.Parallel()

	baseYAML := map[string]any{
		"server": map[string]any{
			"ip":   "127.0.0.1",
			"port": 4444,
		},
	}

	path := tempYAML(t, baseYAML)
	resolver := fixedResolver()
	resolver.ConfigFilePath = path

	input := gen.ConfigInput{}
	m := &mutationResolver{resolver}
	_, err := m.UpdateConfig(context.Background(), input)
	require.NoError(t, err)

	data, err := readConfigFile(path)
	require.NoError(t, err)

	server := data["server"].(map[string]any)
	assert.Equal(t, "127.0.0.1", server["ip"])
	assert.Equal(t, 4444, server["port"])
}

func Test_UpdateConfigMutation_storagePhase2Push_gracefulOnFailure(t *testing.T) {
	t.Parallel()

	baseYAML := map[string]any{
		"storage": map[string]any{
			"postgres": map[string]any{
				"host":     "pg.local",
				"username": "app",
				"port":     5432,
				"database": "hexmagnet",
				"password": "secret_password_7890",
				"ssl_mode": "disable",
			},
			"queue": map[string]any{
				"backend": "memory",
			},
			"search": map[string]any{
				"backend": "postgresql",
			},
		},
	}

	path := tempYAML(t, baseYAML)
	resolver := fixedResolver()
	resolver.Logger = zap.NewNop().Sugar()
	resolver.ConfigFilePath = path
	resolver.QueueRuntime = queue.NewRuntime(resolver.QueueCfg, zap.NewNop().Sugar())
	resolver.SearchRuntime = search.NewRuntime(resolver.SearchCfg)
	resolver.ConfigValidator = stubbedValidator()

	input := gen.ConfigInput{
		Storage: graphql.OmittableOf[*gen.StorageConfigInput](&gen.StorageConfigInput{
			Postgres: graphql.OmittableOf[*gen.PostgresConfigInput](&gen.PostgresConfigInput{
				Host: graphql.OmittableOf[*string](testutil.StrPtr("pg-new.local")),
			}),
		}),
	}

	m := &mutationResolver{resolver}
	result, err := m.UpdateConfig(context.Background(), input)
	require.NoError(t, err, "mutation should succeed even if Phase 2 push fails")

	assert.Equal(t, "pg-new.local", result.Storage.Postgres.Host, "in-memory config should reflect the update")
}

func Test_UpdateConfigMutation_storagePhase2Push_updatesAllRuntimes(t *testing.T) {
	t.Parallel()

	baseYAML := map[string]any{
		"storage": map[string]any{
			"postgres": map[string]any{
				"host":            "pg.local",
				"username":        "app",
				"port":            5432,
				"database":        "hexmagnet",
				"password":        "secret",
				"ssl_mode":        "disable",
				"max_connections": 25,
			},
			"queue": map[string]any{
				"backend": "memory",
			},
			"search": map[string]any{
				"backend": "postgresql",
				"elasticsearch": map[string]any{
					"addresses": []any{"http://es:9200"},
				},
			},
		},
	}

	path := tempYAML(t, baseYAML)
	resolver := fixedResolver()
	resolver.Logger = zap.NewNop().Sugar()
	resolver.ConfigFilePath = path
	resolver.QueueRuntime = queue.NewRuntime(resolver.QueueCfg, zap.NewNop().Sugar())
	resolver.SearchRuntime = search.NewRuntime(resolver.SearchCfg)
	resolver.ConfigValidator = stubbedValidator()

	// Create ConfigManager with subscribers mirroring real gqlfx/module.go wiring.
	cm := configmgr.NewManager(&configmgr.Snapshot{
		Queue:  resolver.QueueCfg,
		Search: resolver.SearchCfg,
	}, "", nil, zap.NewNop().Sugar())
	cm.Subscribe(context.Background(), "queue_test", func(_ context.Context, snap *configmgr.Snapshot) error {
		_ = resolver.QueueRuntime.SwitchTo(snap.Queue) //nolint:contextcheck // SwitchTo accepts no context
		return nil
	}, configmgr.ApplySync)
	cm.Subscribe(context.Background(), "search_test", func(_ context.Context, snap *configmgr.Snapshot) error {
		resolver.SearchRuntime.SwitchBackend(snap.Search)
		return nil
	}, configmgr.ApplySync)
	resolver.ConfigManager = cm

	input := gen.ConfigInput{
		Storage: graphql.OmittableOf[*gen.StorageConfigInput](&gen.StorageConfigInput{
			Queue: graphql.OmittableOf[*gen.QueueConfigInput](&gen.QueueConfigInput{
				Backend: graphql.OmittableOf[*string](testutil.StrPtr("memory")),
			}),
			Search: graphql.OmittableOf[*gen.SearchConfigInput](&gen.SearchConfigInput{
				Backend: graphql.OmittableOf[*string](testutil.StrPtr("elasticsearch")),
			}),
		}),
	}

	m := &mutationResolver{resolver}
	_, err := m.UpdateConfig(context.Background(), input)
	require.NoError(t, err)

	assert.Equal(t, "memory", resolver.QueueRuntime.ActiveBackend.Get(),
		"QueueRuntime backend should be updated by subscriber dispatch")
	assert.Equal(t, "elasticsearch", resolver.SearchRuntime.Backend.Get(),
		"SearchRuntime backend should be updated by subscriber dispatch")
}

func stubbedValidator() *ConfigValidator {
	return &ConfigValidator{
		Postgres: func(context.Context, postgres.Config) error { return nil },
		Elasticsearch: func(context.Context, elasticsearch.Config) error {
			return nil
		},
		KafkaBrokers: func(context.Context, []string) error { return nil },
		TMDB:         func(context.Context, tmdb.Config) error { return nil },
		HTTPEndpoint: func(context.Context, string) error { return nil },
	}
}

func fixedResolver() *Resolver {
	cfg := classifier.NewDefaultConfig()
	cfg.Tmdb = tmdb.Config{
		Enabled:     true,
		AccessToken: "test_token_1234abcd",
		RateLimit:   20,
	}
	cfg.LLM = classifier.LLMConfig{
		Endpoint: "https://llm.local",
		APIKey:   "key_1234567890",
		Model:    "test-model",
		Enabled:  true,
	}

	return &Resolver{
		ServerCfg: servercfg.Config{
			IP:   "127.0.0.1",
			Port: 3333,
			Log: servercfg.LogConfig{
				ConsoleLevel:    "info",
				FileOutputLevel: "off",
				FileRotator: servercfg.FileRotatorConfig{
					MaxBackups: 5,
					Format:     "text",
				},
			},
			EmbedTrackers: []string{},
		},
		DhtCfg: dht.Config{
			Port: 3334,
			Responder: dht.ResponderConfig{
				Enabled:         true,
				GlobalRateLimit: 50,
				PerIPRateLimit:  1,
			},
			BootstrapNodes:               []string{"router.utorrent.com:6881"},
			ReseedBootstrapNodesInterval: time.Minute,
		},
		ClassifierCfg: cfg,
		DHTRequesterCfg: metainforequester.Config{
			RequestLimit:      50,
			RescrapeThreshold: 2592000,
			HashDiscoverLimit: 10,
		},
		PostgresCfg: postgres.Config{
			Host:              "pg.local",
			Username:          "app",
			Port:              5432,
			Database:          "hexmagnet",
			Password:          "secret_password_7890",
			SSLMode:           "disable",
			ConnectionTimeout: 10,
			MaxConnections:    25,
		},
		SearchCfg: indexer.SearchConfig{
			Backend: "postgresql",
			Elasticsearch: elasticsearch.Config{
				Addresses: []string{"http://es:9200"},
				Embedding: embedding.Config{
					Endpoint:           "http://ollama:11434/v1",
					APIKey:             "ollama_key_abcdef1234567890",
					Model:              "bge-m3",
					Dimensions:         1024,
					InstructionEnabled: false,
				},
			},
		},
		QueueCfg: queue.Config{
			Backend: "memory",
			Kafka: kafka.Config{
				Brokers: []string{"localhost:9092"},
			},
		},
	}
}

func Test_applyTorznabInput_maskedAPIKeyKept(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"torznab": map[string]any{
			"api_key": "supersecret",
		},
	}

	masked := maskSecret("supersecret")
	require.NotEqual(t, "supersecret", masked)

	input := gen.TorznabConfigInput{
		APIKey:            graphql.OmittableOf[*string](&masked),
		TrustProxyHeaders: graphql.OmittableOf[*bool](boolPtr(false)),
	}

	require.NoError(t, applyTorznabInput(data, input))

	section := data["torznab"].(map[string]any)
	assert.Equal(t, "supersecret", section["api_key"], "masked key must not clobber the real one")
	assert.Equal(t, false, section["trust_proxy_headers"])
}

func Test_applyTorznabInput_newAPIKeyApplied(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"torznab": map[string]any{
			"api_key": "supersecret",
		},
	}

	input := gen.TorznabConfigInput{
		APIKey: graphql.OmittableOf[*string](strPtr("brand-new-key")),
	}

	require.NoError(t, applyTorznabInput(data, input))

	section := data["torznab"].(map[string]any)
	assert.Equal(t, "brand-new-key", section["api_key"])
}

func Test_applyWebhooksInput_headers(t *testing.T) {
	t.Parallel()

	t.Run("sets headers", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{}

		input := gen.WebhooksConfigInput{
			Headers: graphql.OmittableOf[[]gen.WebhookHeaderInput]([]gen.WebhookHeaderInput{
				{Key: "Authorization", Value: "Bearer tok"},
				{Key: "  ", Value: "ignored"},
			}),
		}

		require.NoError(t, applyWebhooksInput(data, input))

		headers := data["webhooks"].(map[string]any)["headers"].(map[string]any)
		assert.Equal(t, map[string]any{"Authorization": "Bearer tok"}, headers)
	})

	t.Run("masked value keeps existing", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"webhooks": map[string]any{
				"headers": map[string]any{
					"Authorization": "Bearer real-token",
				},
			},
		}

		masked := maskSecret("Bearer real-token")

		input := gen.WebhooksConfigInput{
			Headers: graphql.OmittableOf[[]gen.WebhookHeaderInput]([]gen.WebhookHeaderInput{
				{Key: "Authorization", Value: masked},
				{Key: "X-New", Value: "v"},
			}),
		}

		require.NoError(t, applyWebhooksInput(data, input))

		headers := data["webhooks"].(map[string]any)["headers"].(map[string]any)
		assert.Equal(t, "Bearer real-token", headers["Authorization"])
		assert.Equal(t, "v", headers["X-New"])
	})

	t.Run("omitted headers are removed", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"webhooks": map[string]any{
				"headers": map[string]any{
					"Authorization": "Bearer real-token",
					"X-Old":         "gone",
				},
			},
		}

		masked := maskSecret("Bearer real-token")

		input := gen.WebhooksConfigInput{
			Headers: graphql.OmittableOf[[]gen.WebhookHeaderInput]([]gen.WebhookHeaderInput{
				{Key: "Authorization", Value: masked},
			}),
		}

		require.NoError(t, applyWebhooksInput(data, input))

		headers := data["webhooks"].(map[string]any)["headers"].(map[string]any)
		assert.Equal(t, "Bearer real-token", headers["Authorization"])
		assert.NotContains(t, headers, "X-Old")
	})
}

func Test_configToWebhooks_headersMaskedAndSorted(t *testing.T) {
	t.Parallel()

	cfg := webhook.Config{
		Enabled: true,
		Headers: map[string]string{
			"Authorization": "Bearer secret-token",
			"X-Custom":      "plain",
		},
	}

	out := configToWebhooks(cfg)

	require.Len(t, out.Headers, 2)
	assert.Equal(t, "Authorization", out.Headers[0].Key)
	assert.Equal(t, maskSecret("Bearer secret-token"), out.Headers[0].Value)
	assert.Equal(t, "X-Custom", out.Headers[1].Key)
	assert.Equal(t, maskSecret("plain"), out.Headers[1].Value)
}

func Test_configToTorznab_trustProxyHeaders(t *testing.T) {
	t.Parallel()

	cfg := torznab.NewDefaultConfig()
	assert.True(t, cfg.TrustProxyHeaders)
	assert.True(t, configToTorznab(cfg).TrustProxyHeaders)
}
