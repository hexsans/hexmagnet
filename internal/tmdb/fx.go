package tmdb

import "go.uber.org/fx"

func NewModule() fx.Option {
	return fx.Module(
		"tmdb",
		fx.Provide(
			func(cfg Config) *AccessTokenHolder {
				return NewAccessTokenHolder(cfg.AccessToken)
			},
			New,
			NewHealthCheckFx,
		),
	)
}
