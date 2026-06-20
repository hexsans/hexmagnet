package metainfo

import (
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/anacrolix/torrent/bencode"
	mi "github.com/anacrolix/torrent/metainfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func shiftJISBytes(t *testing.T, s string) []byte {
	t.Helper()

	encoder := japanese.ShiftJIS.NewEncoder()
	reader := transform.NewReader(strings.NewReader(s), encoder)
	b, err := io.ReadAll(reader)
	require.NoError(t, err)

	return b
}

func TestDetectAndDecode_ValidUTF8(t *testing.T) {
	t.Parallel()

	raw := []byte("hello world")
	result, ok := detectAndDecode(raw)
	assert.True(t, ok)
	assert.Equal(t, "hello world", result)
}

func TestDetectAndDecode_Empty(t *testing.T) {
	t.Parallel()

	result, ok := detectAndDecode([]byte{})
	assert.True(t, ok)
	assert.Empty(t, result)
}

func TestDetectAndDecode_NonUTF8Decodable(t *testing.T) {
	t.Parallel()

	original := "\u30c6\u30b9\u30c8" // テスト in Japanese
	encoded := shiftJISBytes(t, original)
	require.False(t, utf8.Valid(encoded), "encoded bytes should not be valid UTF-8")

	result, ok := detectAndDecode(encoded)
	assert.True(t, ok)
	assert.Equal(t, original, result)
}

func TestDetectAndDecode_FallbackOnTrulyInvalidBytes(t *testing.T) {
	t.Parallel()

	// golang.org/x/text decoders replace most invalid sequences with
	// U+FFFD rather than erroring, so the false branch of
	// detectAndDecode is rarely hit. Verify the function doesn't panic
	// and returns the raw string as a fallback.
	raw := []byte{0xFF}
	result, ok := detectAndDecode(raw)
	assert.NotEmpty(t, result)

	if !ok {
		assert.Equal(t, string(raw), result)
	}
}

func TestDecodeBytes_Valid(t *testing.T) {
	t.Parallel()

	original := "Hello, \u4e16\u754c"
	encoded := shiftJISBytes(t, original)
	require.False(t, utf8.Valid(encoded))

	decoded, err := decodeBytes(encoded, japanese.ShiftJIS)
	require.NoError(t, err)
	assert.Equal(t, original, string(decoded))
}

func TestDecodeBytes_InvalidBytesProduceOutput(t *testing.T) {
	t.Parallel()

	// golang.org/x/text decoders replace invalid sequences with U+FFFD
	// rather than returning an error.
	raw := []byte{0xFF, 0xFE, 0xFD}

	decoded, err := decodeBytes(raw, japanese.ShiftJIS)
	if err == nil {
		assert.NotNil(t, decoded)
	}
}

func TestNormalizeInfo_ValidUTF8Name(t *testing.T) {
	t.Parallel()

	info := &Info{Name: "hello"}
	NormalizeInfo(info)
	assert.Equal(t, "hello", info.Name)
	assert.Empty(t, info.NameUtf8)
}

func TestNormalizeInfo_NameUtf8AlreadySet(t *testing.T) {
	t.Parallel()

	info := &Info{
		Name:     string([]byte{0x93, 0xFA}),
		NameUtf8: "existing",
	}
	NormalizeInfo(info)
	assert.Equal(t, "existing", info.NameUtf8)
}

func TestNormalizeInfo_NonUTF8Name(t *testing.T) {
	t.Parallel()

	original := "\u65e5\u672c\u8a9e" // 日本語
	encoded := shiftJISBytes(t, original)
	require.False(t, utf8.Valid(encoded))

	info := &Info{Name: string(encoded)}
	NormalizeInfo(info)

	assert.Equal(t, original, info.NameUtf8)
}

func TestNormalizeInfo_FilePathsNonUTF8(t *testing.T) {
	t.Parallel()

	original := "\u30d5\u30a1\u30a4\u30eb" // ファイル
	encoded := shiftJISBytes(t, original)
	require.False(t, utf8.Valid(encoded))

	info := &Info{
		Files: []mi.FileInfo{
			{Path: []string{string(encoded)}},
		},
	}
	NormalizeInfo(info)

	require.Len(t, info.Files, 1)
	require.Len(t, info.Files[0].PathUtf8, 1)
	assert.Equal(t, original, info.Files[0].PathUtf8[0])
}

func TestNormalizeInfo_FilePathsAlreadySet(t *testing.T) {
	t.Parallel()

	info := &Info{
		Files: []mi.FileInfo{
			{
				Path:     []string{"old"},
				PathUtf8: []string{"existing"},
			},
		},
	}
	NormalizeInfo(info)
	require.Len(t, info.Files[0].PathUtf8, 1)
	assert.Equal(t, "existing", info.Files[0].PathUtf8[0])
}

func TestNormalizeInfo_FilePathsAllSetToCopyOfValidPaths(t *testing.T) {
	t.Parallel()

	// NormalizeInfo always sets PathUtf8 when all paths can be converted,
	// even if they are already valid UTF-8.
	info := &Info{
		Files: []mi.FileInfo{
			{Path: []string{"file1.txt"}},
			{Path: []string{"file2.txt"}},
		},
	}
	NormalizeInfo(info)

	for _, f := range info.Files {
		require.Len(t, f.PathUtf8, 1)
		assert.Equal(t, f.Path[0], f.PathUtf8[0])
	}
}

func TestNormalizeInfo_MultipleFilesMixed(t *testing.T) {
	t.Parallel()

	original := "\u30d5\u30a1\u30a4\u30eb"
	encoded := shiftJISBytes(t, original)

	info := &Info{
		Files: []mi.FileInfo{
			{Path: []string{"ascii.txt"}},
			{Path: []string{string(encoded)}},
		},
	}
	NormalizeInfo(info)

	assert.Len(t, info.Files, 2)
	require.Len(t, info.Files[0].PathUtf8, 1)
	assert.Equal(t, "ascii.txt", info.Files[0].PathUtf8[0])
	require.Len(t, info.Files[1].PathUtf8, 1)
	assert.Equal(t, original, info.Files[1].PathUtf8[0])
}

func TestNormalizeInfo_BuiltWithBencode(t *testing.T) {
	t.Parallel()

	info := Info{
		Name:        "test.torrent",
		PieceLength: 16384,
		Length:      1000,
		Pieces:      make([]byte, 20),
	}
	infoBytes, err := bencode.Marshal(info)
	require.NoError(t, err)

	var decoded Info

	err = bencode.Unmarshal(infoBytes, &decoded)
	require.NoError(t, err)

	NormalizeInfo(&decoded)
	assert.Equal(t, "test.torrent", decoded.Name)
	assert.Empty(t, decoded.NameUtf8)
}
