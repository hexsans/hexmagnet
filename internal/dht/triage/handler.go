package triage

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/hexsans/hexmagnet/internal/blocking"
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/utils"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Action int

const (
	ActionDiscard Action = iota
	ActionGetPeers
	ActionScrape
)

type Result struct {
	Action      Action
	GetPeersMsg *dht.GetPeersMessage
	ScrapeMsg   *dht.ScrapeMessage
}

type Handler interface {
	HandleTriage(ctx context.Context, msg dht.DiscoveredHash) (Result, error)
}

type Params struct {
	fx.In
	Queries         utils.Lazy[*db.Queries]
	BlockingManager utils.Lazy[blocking.Manager]
	Config          dht.Config
	ConfigManager   *configmgr.Manager `optional:"true"`
	Logger          *zap.SugaredLogger
}

type handler struct {
	queries           utils.Lazy[*db.Queries]
	blockingManager   utils.Lazy[blocking.Manager]
	rescrapeThreshold atomic.Uint64
	logger            *zap.SugaredLogger
}

func New(p Params) Handler {
	h := &handler{
		queries:         p.Queries,
		blockingManager: p.BlockingManager,
		logger:          p.Logger.Named("triage"),
	}
	h.rescrapeThreshold.Store(uint64(p.Config.RescrapeThreshold))

	if p.ConfigManager != nil {
		p.ConfigManager.Subscribe(context.Background(), "triage",
			func(_ context.Context, snap *configmgr.Snapshot) error {
				h.rescrapeThreshold.Store(uint64(snap.DHTRequester.RescrapeThreshold))
				return nil
			}, configmgr.ApplyAsync)
	}

	return h
}

func (h *handler) HandleTriage(ctx context.Context, msg dht.DiscoveredHash) (Result, error) {
	id, err := protocol.ParseID(msg.InfoHash)
	if err != nil {
		h.logger.Warnw("failed to parse info hash in triage", "info_hash", msg.InfoHash, "node", msg.Node, "error", err)
		return Result{Action: ActionDiscard}, err
	}

	blockingManager, err := h.blockingManager.Get()
	if err != nil {
		h.logger.Errorw("failed to get blocking manager in triage", "info_hash", msg.InfoHash, "error", err)
		return Result{Action: ActionDiscard}, err
	}

	filtered, filterErr := blockingManager.Filter(ctx, []protocol.ID{id})
	if filterErr != nil {
		return Result{Action: ActionDiscard}, filterErr
	}

	if len(filtered) == 0 {
		return Result{Action: ActionDiscard}, nil
	}

	q, err := h.queries.Get()
	if err != nil {
		return Result{Action: ActionDiscard}, err
	}

	row := q.Read(ctx).QueryRow(ctx, `
		SELECT t.files_count,
			COALESCE(t.seeders, 0), COALESCE(t.leechers, 0),
			t.updated_at
		FROM torrents t
		WHERE t.info_hash = $1
	`, db.FromProtocolID(id))

	var (
		filesCount        *int32
		seeders, leechers int32
		updatedAt         *time.Time
	)

	err = row.Scan(&filesCount, &seeders, &leechers, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{
			Action:      ActionGetPeers,
			GetPeersMsg: &dht.GetPeersMessage{InfoHash: msg.InfoHash, Node: msg.Node},
		}, nil
	}

	if err != nil {
		return Result{}, err
	}

	if filesCount == nil || *filesCount == 0 {
		return Result{
			Action:      ActionGetPeers,
			GetPeersMsg: &dht.GetPeersMessage{InfoHash: msg.InfoHash, Node: msg.Node},
		}, nil
	}

	threshold := time.Duration(h.rescrapeThreshold.Load()) * time.Second
	if (seeders == 0 && leechers == 0) ||
		updatedAt == nil || updatedAt.Before(time.Now().Add(-threshold)) {
		return Result{
			Action:    ActionScrape,
			ScrapeMsg: &dht.ScrapeMessage{InfoHash: msg.InfoHash, Node: msg.Node},
		}, nil
	}

	return Result{Action: ActionDiscard}, nil
}
