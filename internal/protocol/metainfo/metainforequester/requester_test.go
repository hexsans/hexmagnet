package metainforequester

import (
	"bytes"
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/peer_protocol"
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

type mockRequester struct {
	fn func(context.Context, protocol.ID, netip.AddrPort) (Response, error)
}

func (m mockRequester) Request(ctx context.Context, infoHash protocol.ID, addr netip.AddrPort) (Response, error) {
	if m.fn != nil {
		return m.fn(ctx, infoHash, addr)
	}

	return Response{}, nil
}

func TestNewDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := NewDefaultConfig()
	assert.Equal(t, 50, cfg.RequestLimit)
	assert.Equal(t, uint(2592000), cfg.RescrapeThreshold)
	assert.Equal(t, 10, cfg.HashDiscoverLimit)
}

func TestPeerExtensionBits_New(t *testing.T) {
	t.Parallel()

	pex := NewPeerExtensionBits(ExtensionBitDht, ExtensionBitLtep)
	assert.True(t, pex.GetBit(ExtensionBitDht))
	assert.True(t, pex.GetBit(ExtensionBitLtep))
	assert.False(t, pex.GetBit(ExtensionBitFast))
}

func TestPeerExtensionBits_WithBit(t *testing.T) {
	t.Parallel()

	var pex PeerExtensionBits

	pex = pex.WithBit(ExtensionBitDht, true)
	assert.True(t, pex.GetBit(ExtensionBitDht))

	pex = pex.WithBit(ExtensionBitDht, false)
	assert.False(t, pex.GetBit(ExtensionBitDht))
}

func TestPeerExtensionBits_MultipleBits(t *testing.T) {
	t.Parallel()

	pex := NewPeerExtensionBits(ExtensionBitDht, ExtensionBitLtep, ExtensionBitFast)
	assert.True(t, pex.GetBit(ExtensionBitDht))
	assert.True(t, pex.GetBit(ExtensionBitLtep))
	assert.True(t, pex.GetBit(ExtensionBitFast))
}

func TestUintToBigEndian4(t *testing.T) {
	t.Parallel()

	b := uintToBigEndian4(1)
	assert.Equal(t, []byte{0, 0, 0, 1}, b)

	b = uintToBigEndian4(256)
	assert.Equal(t, []byte{0, 0, 1, 0}, b)

	b = uintToBigEndian4(0)
	assert.Equal(t, []byte{0, 0, 0, 0}, b)
}

func TestReadMessage(t *testing.T) {
	t.Parallel()

	data := []byte{0, 0, 0, 5, 1, 2, 3, 4, 5}
	buf := bytes.NewBuffer(data)

	msg, err := readMessage(buf)
	require.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3, 4, 5}, msg)
}

func TestReadMessage_TooLong(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{0x01, 0x00, 0x00, 0x00})

	_, err := readMessage(&buf)
	assert.ErrorContains(t, err, "longer than max allowed")
}

func TestReadMessage_Truncated(t *testing.T) {
	t.Parallel()

	data := []byte{0, 0, 0, 5, 1, 2}
	buf := bytes.NewBuffer(data)

	_, err := readMessage(buf)
	assert.Error(t, err)
}

func TestReadMessage_Empty(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	_, err := readMessage(buf)
	assert.Error(t, err)
}

func TestReadExMessage_SkipsNonExtension(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{0, 0, 0, 3, 19, 1, 2})
	_, _ = buf.Write([]byte{0, 0, 0, 4, 20, 1, 2, 3})

	msg, err := readExMessage(&buf)
	require.NoError(t, err)
	assert.Equal(t, byte(20), msg[0])
	assert.Equal(t, []byte{20, 1, 2, 3}, msg)
}

func TestReadExMessage_ShortResponse(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{0, 0, 0, 1, 20})

	_, err := readExMessage(&buf)
	assert.Error(t, err)
}

func TestReadUmMessage_FiltersNonUTMetadata(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{0, 0, 0, 4, 20, 0, 1, 2})
	_, _ = buf.Write([]byte{0, 0, 0, 4, 20, 1, 3, 4})

	msg, err := readUmMessage(&buf)
	require.NoError(t, err)
	assert.Equal(t, byte(1), msg[1])
	assert.Equal(t, []byte{20, 1, 3, 4}, msg)
}

func TestBtHandshake(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	infoHash := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	clientID := testutil.MustParseID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	response := make([]byte, 68)
	copy(response, peer_protocol.Protocol)

	pex := NewPeerExtensionBits(ExtensionBitLtep)
	copy(response[20:28], pex[:])
	copy(response[28:48], infoHash[:])
	copy(response[48:68], clientID[:])
	_, _ = buf.Write(response)

	hsInfo, err := btHandshake(&buf, infoHash, clientID)
	require.NoError(t, err)
	assert.Equal(t, clientID, hsInfo.PeerID)
	assert.True(t, hsInfo.PeerExtensionBits.GetBit(ExtensionBitLtep))
}

func TestBtHandshake_ProtocolMismatch(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	infoHash := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	clientID := testutil.MustParseID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	response := make([]byte, 68)
	copy(response, []byte("AAAAAAAAAAAAAAAAAAAA"))
	_, _ = buf.Write(response)

	_, err := btHandshake(&buf, infoHash, clientID)
	assert.ErrorContains(t, err, "invalid handshake")
}

func TestBtHandshake_NoLTEP(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	infoHash := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	clientID := testutil.MustParseID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	response := make([]byte, 68)
	copy(response, peer_protocol.Protocol)

	pex := NewPeerExtensionBits()
	copy(response[20:28], pex[:])
	copy(response[28:48], infoHash[:])
	copy(response[48:68], clientID[:])
	_, _ = buf.Write(response)

	_, err := btHandshake(&buf, infoHash, clientID)
	assert.ErrorContains(t, err, "does not support")
}

func TestBtHandshake_InfoHashMismatch(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	infoHash := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	clientID := testutil.MustParseID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	response := make([]byte, 68)
	copy(response, peer_protocol.Protocol)

	pex := NewPeerExtensionBits(ExtensionBitLtep)
	copy(response[20:28], pex[:])
	copy(response[28:48], make([]byte, 20))
	copy(response[48:68], clientID[:])
	_, _ = buf.Write(response)

	_, err := btHandshake(&buf, infoHash, clientID)
	assert.ErrorContains(t, err, "infohash mismatch")
}

func TestExHandshake(t *testing.T) {
	t.Parallel()

	rootDictBytes, marshalErr := bencode.Marshal(rootDict{
		M:            mDict{UTMetadata: 42},
		MetadataSize: 42,
	})
	require.NoError(t, marshalErr)

	msgLen := 2 + len(rootDictBytes)
	extMsg := append([]byte{0x14, 0x00}, rootDictBytes...)

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{
		byte(msgLen >> 24), byte(msgLen >> 16), byte(msgLen >> 8), byte(msgLen),
	})
	_, _ = buf.Write(extMsg)

	metadataSize, utMetadata, err := exHandshake(&buf)
	require.NoError(t, err)
	assert.Equal(t, uint(42), metadataSize)
	assert.Equal(t, uint8(42), utMetadata)
}

func TestExHandshake_NotExtensionHandshake(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{0, 0, 0, 4, 0x14, 0x01, 0, 0})

	_, _, err := exHandshake(&buf)
	assert.ErrorContains(t, err, "first extension message is not")
}

func TestExHandshake_ZeroMetadataSize(t *testing.T) {
	t.Parallel()

	rootDictBytes, marshalErr := bencode.Marshal(rootDict{
		M:            mDict{UTMetadata: 1},
		MetadataSize: 0,
	})
	require.NoError(t, marshalErr)

	msgLen := 2 + len(rootDictBytes)

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{
		byte(msgLen >> 24), byte(msgLen >> 16), byte(msgLen >> 8), byte(msgLen),
		0x14, 0x00,
	})
	_, _ = buf.Write(rootDictBytes)

	_, _, err := exHandshake(&buf)
	assert.ErrorContains(t, err, "metadata too big")
}

func TestExHandshake_BigMetadataSize(t *testing.T) {
	t.Parallel()

	rootDictBytes, marshalErr := bencode.Marshal(rootDict{
		M:            mDict{UTMetadata: 1},
		MetadataSize: maxMetadataSize + 1,
	})
	require.NoError(t, marshalErr)

	msgLen := 2 + len(rootDictBytes)

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{
		byte(msgLen >> 24), byte(msgLen >> 16), byte(msgLen >> 8), byte(msgLen),
		0x14, 0x00,
	})
	_, _ = buf.Write(rootDictBytes)

	_, _, err := exHandshake(&buf)
	assert.ErrorContains(t, err, "metadata too big")
}

func TestRequestAllPieces(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := requestAllPieces(&buf, 2*16*1024, 42)
	require.NoError(t, err)

	assert.Positive(t, buf.Len())
}

func TestRequestAllPieces_NoPieces(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := requestAllPieces(&buf, 0, 1)
	require.NoError(t, err)

	assert.Zero(t, buf.Len())
}

func TestReadAllPieces(t *testing.T) {
	t.Parallel()

	const metadataSize = 100

	extDictBytes, marshalErr := bencode.Marshal(extDict{MsgType: 1, Piece: 0})
	require.NoError(t, marshalErr)

	metadataPiece := make([]byte, metadataSize)
	for i := range metadataPiece {
		metadataPiece[i] = byte(i)
	}

	msgLen := 2 + len(extDictBytes) + len(metadataPiece)

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{
		byte(msgLen >> 24), byte(msgLen >> 16), byte(msgLen >> 8), byte(msgLen),
		0x14, 0x01,
	})
	_, _ = buf.Write(extDictBytes)
	_, _ = buf.Write(metadataPiece)

	pieces, err := readAllPieces(&buf, metadataSize)
	require.NoError(t, err)
	assert.Equal(t, metadataPiece, pieces)
}

func TestReadAllPieces_Rejected(t *testing.T) {
	t.Parallel()

	extDictBytes, marshalErr := bencode.Marshal(extDict{MsgType: 2, Piece: 0})
	require.NoError(t, marshalErr)

	msgLen := 2 + len(extDictBytes)

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{
		byte(msgLen >> 24), byte(msgLen >> 16), byte(msgLen >> 8), byte(msgLen),
		0x14, 0x01,
	})
	_, _ = buf.Write(extDictBytes)

	_, err := readAllPieces(&buf, 100)
	assert.ErrorContains(t, err, "rejected")
}

func TestReadAllPieces_PieceTooLarge(t *testing.T) {
	t.Parallel()

	extDictBytes, marshalErr := bencode.Marshal(extDict{MsgType: 1, Piece: 0})
	require.NoError(t, marshalErr)

	largeData := make([]byte, 20*1024)
	msgLen := 2 + len(extDictBytes) + len(largeData)

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{
		byte(msgLen >> 24), byte(msgLen >> 16), byte(msgLen >> 8), byte(msgLen),
		0x14, 0x01,
	})
	_, _ = buf.Write(extDictBytes)
	_, _ = buf.Write(largeData)

	_, err := readAllPieces(&buf, 1000)
	assert.ErrorContains(t, err, "16kiB")
}

func TestReadAllPieces_ReceivedSizeExceeds(t *testing.T) {
	t.Parallel()

	extDictBytes, marshalErr := bencode.Marshal(extDict{MsgType: 1, Piece: 0})
	require.NoError(t, marshalErr)

	metadataPiece := make([]byte, 16*1024)
	msgLen := 2 + len(extDictBytes) + len(metadataPiece)

	var buf bytes.Buffer

	_, _ = buf.Write([]byte{
		byte(msgLen >> 24), byte(msgLen >> 16), byte(msgLen >> 8), byte(msgLen),
		0x14, 0x01,
	})
	_, _ = buf.Write(extDictBytes)
	_, _ = buf.Write(metadataPiece)

	_, err := readAllPieces(&buf, 1000)
	assert.ErrorContains(t, err, "receivedSize")
}

func TestConnectionSemaphore(t *testing.T) {
	t.Parallel()

	inner := mockRequester{
		fn: func(_ context.Context, _ protocol.ID, _ netip.AddrPort) (Response, error) {
			return Response{}, nil
		},
	}

	cs := &connectionSemaphore{
		requester: inner,
		sem:       semaphore.NewWeighted(1),
	}

	_, err := cs.Request(context.Background(), protocol.ID{}, netip.MustParseAddrPort("1.2.3.4:6881"))
	require.NoError(t, err)
}

func TestConnectionSemaphore_Blocks(t *testing.T) {
	t.Parallel()

	inner := mockRequester{
		fn: func(_ context.Context, _ protocol.ID, _ netip.AddrPort) (Response, error) {
			return Response{}, nil
		},
	}

	cs := &connectionSemaphore{
		requester: inner,
		sem:       semaphore.NewWeighted(1),
	}

	_ = cs.sem.Acquire(context.Background(), 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, err := cs.Request(ctx, protocol.ID{}, netip.MustParseAddrPort("1.2.3.4:6881"))
	assert.Error(t, err)
}

func TestRequestLimiter_PassesThrough(t *testing.T) {
	t.Parallel()

	var called bool

	inner := mockRequester{
		fn: func(_ context.Context, _ protocol.ID, _ netip.AddrPort) (Response, error) {
			called = true
			return Response{}, nil
		},
	}

	rl := requestLimiter{
		requester: inner,
		limiter:   concurrency.NewKeyedLimiter(rate.Inf, 1, 100, time.Minute),
	}

	_, err := rl.Request(context.Background(), protocol.ID{}, netip.MustParseAddrPort("1.2.3.4:6881"))
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRequestLimiter_Blocks(t *testing.T) {
	t.Parallel()

	rl := requestLimiter{
		requester: mockRequester{},
		limiter:   concurrency.NewKeyedLimiter(1, 1, 100, time.Minute),
	}

	addr := netip.MustParseAddrPort("1.2.3.4:6881")

	_, err := rl.Request(context.Background(), protocol.ID{}, addr)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, err = rl.Request(ctx, protocol.ID{}, addr)
	assert.Error(t, err)
}
