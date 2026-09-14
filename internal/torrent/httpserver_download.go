package torrent

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/anacrolix/torrent/bencode"
	"github.com/gin-gonic/gin"
	"github.com/hexsans/hexmagnet/internal/configmgr"
	"github.com/hexsans/hexmagnet/internal/httpserver"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo"
	"github.com/hexsans/hexmagnet/internal/servercfg"
	"github.com/hexsans/hexmagnet/internal/torrentstore"
	"go.uber.org/zap"
)

func New(logger *zap.SugaredLogger, store *torrentstore.Store, cm *configmgr.Manager, config servercfg.Config) httpserver.Option {
	b := &builder{
		logger: logger.Named("torrent_http"),
		store:  store,
	}
	b.embedTrackers.Store(&config.EmbedTrackers)

	if cm != nil {
		cm.Subscribe("embed_trackers",
			func(_ context.Context, snap *configmgr.Snapshot) error {
				b.embedTrackers.Store(&snap.Server.EmbedTrackers)
				return nil
			}, configmgr.ApplyAsync)
	}

	return b
}

type builder struct {
	logger        *zap.SugaredLogger
	store         *torrentstore.Store
	embedTrackers atomic.Pointer[[]string]
}

func (*builder) Key() string {
	return "torrent_download"
}

func (b *builder) Apply(e *gin.Engine) error {
	e.GET("/api/torrents/:infoHash/download", b.handleDownload)
	return nil
}

func (b *builder) handleDownload(c *gin.Context) {
	hashParam := c.Param("infoHash")
	if len(hashParam) != 40 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	hashBytes, decodeErr := hex.DecodeString(hashParam)
	if decodeErr != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	var infoHash protocol.ID
	copy(infoHash[:], hashBytes)

	if b.store == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	data, storeErr := b.store.Get(c.Request.Context(), infoHash)
	if errors.Is(storeErr, torrentstore.ErrNotFound) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if storeErr != nil {
		b.logger.Errorw("failed to read torrent file from disk", "info_hash", hashParam, "error", storeErr)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	var tf metainfo.TorrentFile
	if unmarshalErr := bencode.Unmarshal(data, &tf); unmarshalErr != nil {
		b.logger.Errorw("failed to unmarshal torrent file from disk", "info_hash", hashParam, "error", unmarshalErr)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	outer := metainfo.TorrentFile{
		Info:         tf.Info,
		CreationDate: tf.CreationDate,
	}

	if trackers := b.embedTrackers.Load(); trackers != nil && len(*trackers) > 0 {
		t := *trackers
		if len(outer.AnnounceList) > 0 {
			outer.AnnounceList = append(outer.AnnounceList, t)
		} else {
			outer.AnnounceList = [][]string{t}
		}

		if outer.Announce == "" {
			outer.Announce = t[0]
		}
	}

	outerBytes, marshalErr := bencode.Marshal(outer)
	if marshalErr != nil {
		b.logger.Errorw("failed to marshal torrent file", "error", marshalErr)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	filename := fmt.Sprintf("%s.torrent", tf.Info.BestName())

	c.Header("Content-Type", "application/x-bittorrent")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/x-bittorrent", outerBytes)
}
