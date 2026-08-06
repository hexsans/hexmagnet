package version

import "go.uber.org/fx"

func NewModule() fx.Option {
	return fx.Module(
		"version",
		fx.Provide(fx.Annotated{
			Name: "app_version",
			Target: func() string {
				v := GitTagValue()
				if v == "" {
					v = "unknown"
				}

				return v
			},
		}),
		fx.Provide(NewHealthCheck),
	)
}
