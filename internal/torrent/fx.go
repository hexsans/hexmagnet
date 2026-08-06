package torrent

import (
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/httpserver"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/torrentstore"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type newModuleParams struct {
	fx.In
	Logger        *zap.SugaredLogger
	Store         *torrentstore.Store
	ConfigManager *configmgr.Manager `optional:"true"`
	Config        servercfg.Config
}

func NewModule() fx.Option {
	return fx.Module(
		"torrent_download",
		fx.Provide(
			fx.Annotate(
				func(p newModuleParams) httpserver.Option {
					return New(p.Logger, p.Store, p.ConfigManager, p.Config)
				},
				fx.ResultTags(`group:"http_server_options"`),
			),
		),
	)
}
