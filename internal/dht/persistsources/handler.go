package persistsources

import (
	"context"
	"errors"

	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Handler interface {
	HandlePersistSources(ctx context.Context, msg dht.ScrapeResultMessage) error
}

type Params struct {
	fx.In
	Queries utils.Lazy[*db.Queries]
	Logger  *zap.SugaredLogger
}

type handler struct {
	queries utils.Lazy[*db.Queries]
	logger  *zap.SugaredLogger
}

func New(p Params) Handler {
	return &handler{queries: p.Queries, logger: p.Logger.Named("persistsources")}
}

func (h *handler) HandlePersistSources(ctx context.Context, msg dht.ScrapeResultMessage) error {
	id, err := dht.ParseInfoHash(h.logger, msg.InfoHash, "persist sources")
	if err != nil {
		return err
	}

	q, err := h.queries.Get()
	if err != nil {
		h.logger.Errorw("failed to get queries in persist sources", "info_hash", msg.InfoHash, "error", err)
		return err
	}

	_, err = q.GetTorrent(ctx, db.FromProtocolID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return err
	}

	return q.UpdateTorrentSeeders(ctx, db.UpdateTorrentSeedersParams{
		InfoHash: db.FromProtocolID(id),
		Seeders:  db.FromUintToInt32Ptr(msg.Seeders),
		Leechers: db.FromUintToInt32Ptr(msg.Leechers),
	})
}
