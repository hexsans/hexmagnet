package app

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/hexsans/hexmagnet/internal/app/appfx"
	"github.com/hexsans/hexmagnet/internal/classifier"
	"github.com/hexsans/hexmagnet/internal/config"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"github.com/hexsans/hexmagnet/internal/logging"
	"github.com/hexsans/hexmagnet/internal/processor/enrich/indexer"
	"github.com/hexsans/hexmagnet/internal/protocol/dht"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/retryqueue"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/torznab"
	"github.com/hexsans/hexmagnet/internal/webhook"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New() *fx.App {
	resolved, err := config.Load(
		config.SpecEntry{
			Key: "server", DefaultValue: servercfg.NewDefaultConfig(),
			ValidatorOpts: []config.ValidatorOption{
				config.WithStructValidator(func(sl validator.StructLevel) {
					lc := sl.Current().Interface().(servercfg.LogConfig)
					if lc.FileOutputLevel != "off" && lc.FileRotator.Path == "" {
						sl.ReportError(lc.FileRotator.Path, "FileRotator.Path", "Path", "required", "")
					}
				}, servercfg.LogConfig{}),
				config.WithStructValidator(servercfg.ValidateTorrentFilePath, servercfg.Config{}),
			},
		},
		config.SpecEntry{Key: "dht", DefaultValue: dht.NewDefaultConfig()},
		config.SpecEntry{Key: "storage.search", DefaultValue: indexer.DefaultSearchConfig()},
		config.SpecEntry{Key: "storage.postgres", DefaultValue: postgres.NewDefaultConfig()},
		config.SpecEntry{Key: "storage.queue", DefaultValue: queue.NewDefaultConfig()},
		config.SpecEntry{Key: "classifier", DefaultValue: classifier.NewDefaultConfig()},
		config.SpecEntry{Key: "dht.requester", DefaultValue: metainforequester.NewDefaultConfig()},
		config.SpecEntry{Key: "torznab", DefaultValue: torznab.NewDefaultConfig()},
		config.SpecEntry{Key: "webhooks", DefaultValue: webhook.NewDefaultConfig()},
		config.SpecEntry{Key: "retry_queue", DefaultValue: retryqueue.NewDefaultConfig()},
	)
	if err != nil {
		return fx.New(fx.Error(fmt.Errorf("config error: %w", err)))
	}

	return fx.New(
		fx.RecoverFromPanics(),
		fx.Supply(resolved.Resolved.NodeMap["server"].Value.(servercfg.Config)),
		fx.Supply(resolved.Resolved.NodeMap["dht"].Value.(dht.Config)),
		fx.Supply(resolved.Resolved.NodeMap["storage.search"].Value.(indexer.SearchConfig)),
		fx.Supply(resolved.Resolved.NodeMap["storage.postgres"].Value.(postgres.Config)),
		fx.Supply(resolved.Resolved.NodeMap["storage.queue"].Value.(queue.Config)),
		fx.Supply(resolved.Resolved.NodeMap["classifier"].Value.(classifier.Config)),
		fx.Supply(resolved.Resolved.NodeMap["classifier"].Value.(classifier.Config).Tmdb),
		fx.Supply(resolved.Resolved.NodeMap["dht.requester"].Value.(metainforequester.Config)),
		fx.Supply(resolved.Resolved.NodeMap["torznab"].Value.(torznab.Config)),
		fx.Supply(resolved.Resolved.NodeMap["webhooks"].Value.(webhook.Config)),
		fx.Supply(resolved.Resolved.NodeMap["retry_queue"].Value.(retryqueue.Config)),
		fx.Supply(*resolved.Resolved),
		fx.Supply(resolved.Path),
		appfx.New(resolved.Resolved.NodeMap["storage.queue"].Value.(queue.Config)),
		logging.WithLogger(),
		fx.Invoke(func(
			logger *zap.SugaredLogger,
		) {
			logger.Infow("app starting")
		}),
	)
}
