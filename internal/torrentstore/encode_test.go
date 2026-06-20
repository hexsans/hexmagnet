package torrentstore

import (
	"testing"
	"time"

	"github.com/anacrolix/torrent/bencode"
	mi "github.com/anacrolix/torrent/metainfo"
	"github.com/hexsans/hexmagnet/internal/protocol/metainfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeTorrentFile(t *testing.T) {
	t.Parallel()

	rawInfo, err := bencode.Marshal(mi.Info{
		Name:   "example.mkv",
		Length: 12345,
	})
	require.NoError(t, err)

	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	data, err := EncodeTorrentFile(rawInfo, createdAt)
	require.NoError(t, err)

	var tf metainfo.TorrentFile
	require.NoError(t, bencode.Unmarshal(data, &tf))

	assert.Equal(t, "example.mkv", tf.Info.Name)
	assert.Equal(t, int64(12345), tf.Info.Length)
	assert.Equal(t, createdAt.Unix(), tf.CreationDate)
	assert.Empty(t, tf.Announce)
	assert.Nil(t, tf.AnnounceList)
}

func TestEncodeTorrentFileInvalidRaw(t *testing.T) {
	t.Parallel()

	_, err := EncodeTorrentFile([]byte("not bencode"), time.Now())
	assert.Error(t, err)
}
