package tmdb

import (
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ConfigUpdater applies runtime config updates to the TMDB client.
type ConfigUpdater interface {
	UpdateConfig(cfg Config)
}

type Params struct {
	fx.In
	Config            Config
	AccessTokenHolder *AccessTokenHolder
	Logger            *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Client        utils.Lazy[Client]
	ConfigUpdater ConfigUpdater
}

func New(p Params) Result {
	if p.Config.Enabled && p.Config.AccessToken == "" {
		p.Logger.Warn("TMDB enabled but no access token — requests will be silently skipped")
	}

	rl := &requesterLazy{
		config:            p.Config,
		logger:            p.Logger,
		accessTokenHolder: p.AccessTokenHolder,
	}

	return Result{
		Client: utils.NewLazy(func() (Client, error) {
			return client{requester: rl}, nil
		}),
		ConfigUpdater: rl,
	}
}
