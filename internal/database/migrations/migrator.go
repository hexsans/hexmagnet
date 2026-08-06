package migrations

import (
	"context"
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	migrationssql "github.com/hexsans/hexmagnet/database/migrations"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	SQLDB  utils.Lazy[*sql.DB]
	Logger *zap.SugaredLogger
}

type Result struct {
	fx.Out
	Migrator utils.Lazy[Migrator]
}

func New(p Params) Result {
	return Result{
		Migrator: utils.NewLazy(func() (Migrator, error) {
			db, err := p.SQLDB.Get()
			if err != nil {
				return nil, err
			}

			return &migrator{
				db:     db,
				logger: p.Logger.Named("migrator"),
			}, nil
		}),
	}
}

type Migrator interface {
	Up(ctx context.Context) error
	UpTo(ctx context.Context, version int64) error
	Down(ctx context.Context) error
	DownTo(ctx context.Context, version int64) error
}

type migrator struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

func (m *migrator) newMigrate() (*migrate.Migrate, error) {
	src, err := iofs.New(migrationssql.FS, ".")
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(m.db, &postgres.Config{})
	if err != nil {
		return nil, err
	}

	migrator, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return nil, err
	}

	return migrator, nil
}

func (m *migrator) Up(_ context.Context) error {
	migrator, err := m.newMigrate()
	if err != nil {
		return err
	}
	defer migrator.Close()

	m.logger.Info("checking and applying migrations...")

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func (m *migrator) UpTo(_ context.Context, version int64) error {
	migrator, err := m.newMigrate()
	if err != nil {
		return err
	}
	defer migrator.Close()

	if err := migrator.Migrate(uint(version)); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func (m *migrator) Down(_ context.Context) error {
	migrator, err := m.newMigrate()
	if err != nil {
		return err
	}
	defer migrator.Close()

	if err := migrator.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func (m *migrator) DownTo(_ context.Context, version int64) error {
	migrator, err := m.newMigrate()
	if err != nil {
		return err
	}
	defer migrator.Close()

	if err := migrator.Migrate(uint(version)); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
