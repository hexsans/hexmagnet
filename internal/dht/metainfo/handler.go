package metainfo

import (
	"context"
	"errors"
	"net/netip"
	"regexp"
	"sync"
	"time"

	"github.com/hexsans/hexmagnet/internal/blocking"
	"github.com/hexsans/hexmagnet/internal/database/db"
	"github.com/hexsans/hexmagnet/internal/dht"
	"github.com/hexsans/hexmagnet/internal/dhtcrawler"
	"github.com/hexsans/hexmagnet/internal/model"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/client"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/banning"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo/metainforequester"
	"github.com/hexsans/hexmagnet/internal/torrentstore"
	"github.com/hexsans/hexmagnet/internal/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Handler interface {
	HandleGetPeers(ctx context.Context, msg dht.GetPeersMessage) (*dht.MetaInfoMessage, error)
}

type Params struct {
	fx.In
	Client            utils.Lazy[client.Client]
	MetainfoRequester metainforequester.Requester
	BanningChecker    banning.Checker `name:"metainfo_banning_checker"`
	KTable            ktable.Table
	BlockingManager   utils.Lazy[blocking.Manager]
	Queries           utils.Lazy[*db.Queries]
	Store             *torrentstore.Store
	Runtime           *dhtcrawler.Runtime `name:"dht_crawler_runtime" optional:"true"`
	Logger            *zap.SugaredLogger
}

type handler struct {
	client            utils.Lazy[client.Client]
	metainfoRequester metainforequester.Requester
	banningChecker    banning.Checker
	kTable            ktable.Table
	blockingManager   utils.Lazy[blocking.Manager]
	queries           utils.Lazy[*db.Queries]
	store             *torrentstore.Store
	runtime           *dhtcrawler.Runtime
	logger            *zap.SugaredLogger
}

func New(p Params) Handler {
	return &handler{
		client:            p.Client,
		metainfoRequester: p.MetainfoRequester,
		banningChecker:    p.BanningChecker,
		kTable:            p.KTable,
		blockingManager:   p.BlockingManager,
		queries:           p.Queries,
		store:             p.Store,
		runtime:           p.Runtime,
		logger:            p.Logger,
	}
}

func (h *handler) HandleGetPeers(ctx context.Context, msg dht.GetPeersMessage) (*dht.MetaInfoMessage, error) {
	id, err := protocol.ParseID(msg.InfoHash)
	if err != nil {
		return nil, err
	}

	cl, err := h.client.Get()
	if err != nil {
		return nil, err
	}

	blockingManager, err := h.blockingManager.Get()
	if err != nil {
		return nil, err
	}

	node, err := netip.ParseAddrPort(msg.Node)
	if err != nil {
		return nil, err
	}

	peers, err := h.discoverPeers(ctx, cl, node, id)
	if errors.Is(err, errNoPeersFound) {
		return nil, nil //nolint:nilnil // no peers for this hash is a normal outcome; the consumer skips nil results
	}

	if err != nil {
		h.logger.Debugw("failed to discover peers", "info_hash", msg.InfoHash, "error", err)
		return nil, err
	}

	mi, err := h.fetchMetaInfo(ctx, blockingManager, id, peers)
	if err != nil {
		h.logger.Infow("all peers failed to return metainfo, skipping info_hash", "info_hash", msg.InfoHash)
		return nil, err
	}

	var totalSize uint64

	files := make([]dht.MetainfoFile, 0, len(mi.Info.Files))
	for _, f := range mi.Info.Files {
		totalSize += uint64(f.Length)
		files = append(files, dht.MetainfoFile{
			PathParts: f.BestPath(),
			Size:      uint64(f.Length),
		})
	}

	if totalSize == 0 {
		totalSize = uint64(mi.Info.TotalLength())
	}

	private := false
	if mi.Info.Private != nil {
		private = *mi.Info.Private
	}

	if err := h.persistTorrent(ctx, id, mi.RawInfoBytes, mi.Info.BestName(), private, files, totalSize); err != nil {
		h.logger.Errorw("failed to persist torrent", "info_hash", msg.InfoHash, "error", err)
		return nil, err
	}

	if h.runtime != nil {
		h.runtime.PushActivity("metadata", mi.Info.BestName())
	}

	return &dht.MetaInfoMessage{
		InfoHash:  msg.InfoHash,
		Name:      mi.Info.BestName(),
		Private:   private,
		Files:     files,
		TotalSize: totalSize,
		Node:      msg.Node,
	}, nil
}

func (h *handler) discoverPeers(ctx context.Context, cl client.Client, node netip.AddrPort, id protocol.ID) ([]netip.AddrPort, error) {
	res, err := cl.GetPeers(ctx, node, id)
	if err != nil {
		return nil, err
	}

	for _, n := range res.Nodes {
		h.kTable.BatchCommand(ktable.PutNode{
			ID:      n.ID,
			Addr:    n.Addr,
			Options: []ktable.NodeOption{ktable.NodeResponded()},
		})
	}

	if len(res.Values) == 0 {
		return nil, errNoPeersFound
	}

	return res.Values, nil
}

var errNoPeersFound = errors.New("no peers found")

func (h *handler) fetchMetaInfo(
	ctx context.Context,
	blockingManager blocking.Manager,
	hash protocol.ID,
	peers []netip.AddrPort,
) (metainforequester.Response, error) {
	var errs []error

	errsMutex := sync.Mutex{}
	addErr := func(err error) {
		errsMutex.Lock()
		defer errsMutex.Unlock()

		errs = append(errs, err)
	}

	maxPeers := min(5, len(peers))
	for i := range maxPeers {
		res, reqErr := h.metainfoRequester.Request(ctx, hash, peers[i])
		if reqErr != nil {
			h.logger.Debugw("failed to request metainfo from peer", "info_hash", hash.String(), "addr", peers[i].String(), "error", reqErr)
			addErr(reqErr)

			continue
		}

		if banErr := h.banningChecker.Check(res.Info); banErr != nil {
			if blockErr := blockingManager.Block(ctx, hash, banErr.Error()); blockErr != nil {
				addErr(blockErr)
			}

			return metainforequester.Response{}, banErr
		}

		return res, nil
	}

	return metainforequester.Response{}, errors.Join(errs...)
}

func (h *handler) persistTorrent(
	ctx context.Context,
	id protocol.ID,
	rawInfoBytes []byte,
	name string,
	private bool,
	files []dht.MetainfoFile,
	totalSize uint64,
) error {
	q, err := h.queries.Get()
	if err != nil {
		return err
	}

	var (
		filesCount model.NullUint
		modelFiles []model.TorrentFile
		size       uint64
	)

	if len(files) == 0 {
		size = totalSize
		pathParts := []string{name}
		modelFiles = []model.TorrentFile{{
			InfoHash:  id,
			Index:     0,
			PathParts: pathParts,
			Size:      size,
		}}
		filesCount = model.NewNullUint(1)
	} else {
		modelFiles = make([]model.TorrentFile, 0, len(files))

		var realFileCount int

		for i, f := range files {
			if isPaddingFile(f.PathParts) {
				continue
			}

			realFileCount++
			size += f.Size

			modelFiles = append(modelFiles, model.TorrentFile{
				InfoHash:  id,
				Index:     uint(i),
				PathParts: f.PathParts,
				Size:      f.Size,
			})
		}

		if realFileCount > 0 {
			filesCount = model.NewNullUint(uint(realFileCount))
		}
	}

	torrent := model.Torrent{
		InfoHash:   id,
		Name:       name,
		Size:       size,
		Private:    private,
		Files:      modelFiles,
		FilesCount: filesCount,
	}

	tx, err := q.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tq := q.WithTx(tx)

	if err := tq.UpsertTorrent(ctx, db.TorrentToUpsertParams(torrent)); err != nil {
		return err
	}

	for _, f := range torrent.Files {
		if err := tq.UpsertTorrentFile(ctx, db.TorrentFileToUpsertParams(f)); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	h.persistTorrentFile(ctx, id, rawInfoBytes, time.Now())

	return nil
}

// persistTorrentFile writes the torrent file to disk. It is best-effort:
// failures are logged but do not fail the pipeline.
func (h *handler) persistTorrentFile(ctx context.Context, id protocol.ID, rawInfoBytes []byte, createdAt time.Time) {
	if h.store == nil || len(rawInfoBytes) == 0 {
		return
	}

	data, err := torrentstore.EncodeTorrentFile(rawInfoBytes, createdAt)
	if err != nil {
		h.logger.Warnw("failed to encode torrent file", "info_hash", id.String(), "error", err)
		return
	}

	if err := h.store.Put(ctx, id, data); err != nil {
		h.logger.Warnw("failed to write torrent file", "info_hash", id.String(), "error", err)
	}
}

var paddingFileRe = regexp.MustCompile(`^_____padding_file_\d+_.*____$`)

func isPaddingFile(pathParts []string) bool {
	if len(pathParts) == 0 {
		return false
	}

	return paddingFileRe.MatchString(pathParts[len(pathParts)-1])
}
