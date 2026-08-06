package metainfo

import (
	"testing"

	"github.com/anacrolix/torrent/bencode"
	mi "github.com/anacrolix/torrent/metainfo"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMetaInfoBytes_Valid(t *testing.T) {
	t.Parallel()

	info := Info{
		Name:        "test.torrent",
		PieceLength: 16384,
		Length:      1000,
		Pieces:      make([]byte, 20),
	}
	infoBytes, err := bencode.Marshal(info)
	require.NoError(t, err)

	infoHash := protocol.ID(mi.HashBytes(infoBytes))
	result, err := ParseMetaInfoBytes(infoHash, infoBytes)
	require.NoError(t, err)

	assert.Equal(t, "test.torrent", result.Name)
	assert.Equal(t, int64(16384), result.PieceLength)
	assert.Equal(t, int64(1000), result.Length)
	assert.Equal(t, info.Pieces, result.Pieces)
}

func TestParseMetaInfoBytes_WrongHash(t *testing.T) {
	t.Parallel()

	info := Info{
		Name:        "test.torrent",
		PieceLength: 16384,
		Length:      1000,
		Pieces:      make([]byte, 20),
	}
	infoBytes, err := bencode.Marshal(info)
	require.NoError(t, err)

	wrongHash := protocol.ID{
		0x01,
		0x02,
		0x03,
		0x04,
		0x05,
		0x06,
		0x07,
		0x08,
		0x09,
		0x0a,
		0x0b,
		0x0c,
		0x0d,
		0x0e,
		0x0f,
		0x10,
		0x11,
		0x12,
		0x13,
		0x14,
	}
	_, err = ParseMetaInfoBytes(wrongHash, infoBytes)
	assert.ErrorContains(t, err, "wrong hash")
}

func TestParseMetaInfoBytes_InvalidBencode(t *testing.T) {
	t.Parallel()

	badBytes := []byte("not valid bencode")
	infoHash := protocol.ID(mi.HashBytes(badBytes))
	_, err := ParseMetaInfoBytes(infoHash, badBytes)
	assert.ErrorContains(t, err, "error unmarshaling")
}

func TestParseMetaInfoBytes_WrongHashAndInvalidBencode(t *testing.T) {
	t.Parallel()

	_, err := ParseMetaInfoBytes(protocol.ID{}, []byte("garbage"))
	assert.ErrorContains(t, err, "wrong hash")
}

func TestParseMetaInfoBytes_NormalizesInfo(t *testing.T) {
	t.Parallel()

	info := Info{
		Name:        "test.torrent",
		PieceLength: 16384,
		Length:      1000,
		Pieces:      make([]byte, 20),
	}

	infoBytes, err := bencode.Marshal(info)
	require.NoError(t, err)

	infoHash := protocol.ID(mi.HashBytes(infoBytes))
	result, err := ParseMetaInfoBytes(infoHash, infoBytes)
	require.NoError(t, err)
	assert.Equal(t, "test.torrent", result.Name)
}

func TestParseMetaInfoBytes_MultipleFiles(t *testing.T) {
	t.Parallel()

	info := Info{
		Name:        "multi",
		PieceLength: 16384,
		Pieces:      make([]byte, 20),
		Files: []mi.FileInfo{
			{Length: 100, Path: []string{"dir", "file1.txt"}},
			{Length: 200, Path: []string{"dir", "file2.txt"}},
		},
	}

	infoBytes, err := bencode.Marshal(info)
	require.NoError(t, err)

	infoHash := protocol.ID(mi.HashBytes(infoBytes))
	result, err := ParseMetaInfoBytes(infoHash, infoBytes)
	require.NoError(t, err)

	require.Len(t, result.Files, 2)
	assert.Equal(t, int64(100), result.Files[0].Length)
	assert.Equal(t, []string{"dir", "file1.txt"}, result.Files[0].Path)
	assert.Equal(t, int64(200), result.Files[1].Length)
	assert.Equal(t, []string{"dir", "file2.txt"}, result.Files[1].Path)
}

func TestParseMetaInfoBytes_PrivateFlag(t *testing.T) {
	t.Parallel()

	private := true
	info := Info{
		Name:        "private.torrent",
		PieceLength: 16384,
		Length:      1000,
		Pieces:      make([]byte, 20),
		Private:     &private,
	}

	infoBytes, err := bencode.Marshal(info)
	require.NoError(t, err)

	infoHash := protocol.ID(mi.HashBytes(infoBytes))
	result, err := ParseMetaInfoBytes(infoHash, infoBytes)
	require.NoError(t, err)

	require.NotNil(t, result.Private)
	assert.True(t, *result.Private)
}
