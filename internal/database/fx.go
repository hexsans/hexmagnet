package database

import (
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/database/migrations"
	"github.com/hexsans/hexmagnet/internal/database/postgres"
	"go.uber.org/fx"
)

func NewModule() fx.Option {
	return fx.Module(
		"database",
		fx.Provide(
			NewHealthCheck,
			migrations.New,
			postgres.New,
			db.NewFx,
		),
	)
}
