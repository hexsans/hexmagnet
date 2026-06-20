package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Runtime struct {
	poolRef *PoolRef
	cfg     Config
	onPool  func(*pgxpool.Pool)
}

func (r *Runtime) SetOnPoolReplaced(fn func(*pgxpool.Pool)) {
	r.onPool = fn
}

func (r *Runtime) PoolRef() *PoolRef {
	return r.poolRef
}

func (r *Runtime) GetPool() *pgxpool.Pool {
	return r.poolRef.Get()
}

func (r *Runtime) GetConfig() Config {
	return r.cfg
}

// Reconnect creates a new pool with the given config and atomically swaps it.
// Returns the old pool so the caller can close it after a grace period.
func (r *Runtime) Reconnect(ctx context.Context, cfg Config, logger *zap.SugaredLogger) (*pgxpool.Pool, error) {
	newPool, err := newPool(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("reconnect: %w", err)
	}

	r.cfg = cfg

	oldPool := r.poolRef.Replace(newPool)
	if r.onPool != nil {
		r.onPool(oldPool)
	}

	return oldPool, nil
}

// ValidateConnection attempts to connect to the database with the given
// config and returns an error if it cannot. The pool is closed before
// returning and no state is mutated.
func ValidateConnection(ctx context.Context, cfg Config, logger *zap.SugaredLogger) error {
	pool, err := newPool(ctx, cfg, logger)
	if err != nil {
		return err
	}

	pool.Close()

	return nil
}

func newPool(ctx context.Context, cfg Config, logger *zap.SugaredLogger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.CreateDSN())
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxConnections)
	poolCfg.MaxConnLifetime = 15 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	poolCfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "tsvector",
			OID:   pgtype.TSVectorOID,
			Codec: &pgtype.TextCodec{},
		})

		return nil
	}

	pl, plErr := pgxpool.NewWithConfig(ctx, poolCfg)
	if plErr != nil {
		return nil, plErr
	}

	if pingErr := waitForPing(ctx, logger, pl); pingErr != nil {
		pl.Close()
		return nil, pingErr
	}

	return pl, nil
}
