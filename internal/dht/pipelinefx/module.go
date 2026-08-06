package pipelinefx

import (
	"time"

	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/dht/metainfo"
	"github.com/hexsans/hexmagnet/internal/dht/persist"
	"github.com/hexsans/hexmagnet/internal/dht/persistsources"
	"github.com/hexsans/hexmagnet/internal/dht/scrape"
	"github.com/hexsans/hexmagnet/internal/dht/triage"
	"github.com/hexsans/hexmagnet/internal/dhtcrawler"
	"go.uber.org/fx"
)

type ResponderConfig struct {
	Enabled         bool
	GlobalRateLimit int
	PerIPRateLimit  int
}

type Config struct {
	Port                         uint16
	Responder                    ResponderConfig
	BootstrapNodes               []string
	ReseedBootstrapNodesInterval time.Duration
}

func New() fx.Option {
	return fx.Module(
		"dht_pipeline",
		fx.Provide(
			func(cfg dhtcrawler.Config) dht.Config {
				return dht.Config{
					RescrapeThreshold: cfg.RescrapeThreshold,
				}
			},
			triage.New,
			triage.NewQueueConsumer,

			metainfo.New,
			metainfo.NewQueueConsumer,

			scrape.New,
			scrape.NewQueueConsumer,

			persist.New,
			persist.NewQueueConsumer,

			persistsources.New,
			persistsources.NewQueueConsumer,
		),
	)
}
