package torrentstore

import (
	"fmt"
	"time"

	"github.com/anacrolix/torrent/bencode"
	mi "github.com/anacrolix/torrent/metainfo"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo"
)

// EncodeTorrentFile wraps the raw bencoded info dictionary into a valid
// .torrent file without any trackers. Trackers are appended at download time.
func EncodeTorrentFile(rawInfo []byte, createdAt time.Time) ([]byte, error) {
	var info mi.Info
	if err := bencode.Unmarshal(rawInfo, &info); err != nil {
		return nil, fmt.Errorf("unmarshal info dict: %w", err)
	}

	outer := metainfo.TorrentFile{
		Info:         info,
		CreationDate: createdAt.Unix(),
	}

	data, err := bencode.Marshal(outer)
	if err != nil {
		return nil, fmt.Errorf("marshal torrent file: %w", err)
	}

	return data, nil
}
