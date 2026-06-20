package scrape

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/dhtcrawler"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/client"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Handler interface {
	HandleScrape(ctx context.Context, msg dht.ScrapeMessage) (*dht.ScrapeResultMessage, error)
}

type Params struct {
	fx.In
	Client  utils.Lazy[client.Client]
	KTable  ktable.Table
	Runtime *dhtcrawler.Runtime `name:"dht_crawler_runtime" optional:"true"`
	Logger  *zap.SugaredLogger
}

type handler struct {
	client  utils.Lazy[client.Client]
	kTable  ktable.Table
	runtime *dhtcrawler.Runtime
	logger  *zap.SugaredLogger
}

func New(p Params) Handler {
	return &handler{
		client:  p.Client,
		kTable:  p.KTable,
		runtime: p.Runtime,
		logger:  p.Logger,
	}
}

func (h *handler) HandleScrape(ctx context.Context, msg dht.ScrapeMessage) (*dht.ScrapeResultMessage, error) {
	id, err := protocol.ParseID(msg.InfoHash)
	if err != nil {
		return nil, err
	}

	cl, err := h.client.Get()
	if err != nil {
		return nil, err
	}

	node, err := netip.ParseAddrPort(msg.Node)
	if err != nil {
		return nil, err
	}

	res, err := cl.GetPeersScrape(ctx, node, id)
	if err != nil {
		if errors.Is(err, client.ErrNoScrapeSupport) {
			h.logger.Debugw("node does not support scrape", "info_hash", msg.InfoHash, "node", msg.Node)
			return nil, nil //nolint:nilnil // nodes without scrape support are skipped silently
		}

		h.logger.Debugw(
			"peer scrape timed out, peer may be unreachable",
			"info_hash",
			msg.InfoHash,
			"node",
			msg.Node,
			"error",
			err,
		)

		return nil, err
	}

	if len(res.Nodes) > 0 {
		for _, n := range res.Nodes {
			h.kTable.BatchCommand(ktable.PutNode{
				ID:      n.ID,
				Addr:    n.Addr,
				Options: []ktable.NodeOption{ktable.NodeResponded()},
			})
		}
	}

	seeders := uint(res.BfSeeders.ApproximatedSize())
	leechers := uint(res.BfPeers.ApproximatedSize())

	return &dht.ScrapeResultMessage{
		InfoHash:  msg.InfoHash,
		Seeders:   seeders,
		Leechers:  leechers,
		ScrapedAt: time.Now(),
	}, nil
}
