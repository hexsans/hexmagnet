package dht

import (
	"net"
	"testing"

	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeInfoString(t *testing.T) {
	t.Parallel()

	id := testutil.MustParseID("0102030405060708090a0b0c0d0e0f1011121314")
	ni := NodeInfo{
		ID:   id,
		Addr: NodeAddr{IP: net.IP{1, 2, 3, 4}, Port: 6881},
	}
	s := ni.String()
	assert.Contains(t, s, "1.2.3.4:6881")
	assert.Contains(t, s, "{")
	assert.Contains(t, s, " at ")
}

func TestNodeInfoMarshalBinary(t *testing.T) {
	t.Parallel()

	id := testutil.MustParseID("0102030405060708090a0b0c0d0e0f1011121314")
	ni := NodeInfo{
		ID:   id,
		Addr: NodeAddr{IP: net.IP{1, 2, 3, 4}, Port: 6881},
	}
	b, err := ni.MarshalBinary()
	require.NoError(t, err)
	assert.Len(t, b, 26)
	assert.Equal(t, byte(0x01), b[0])
	assert.Equal(t, byte(0x14), b[19])
	assert.Equal(t, byte(0x01), b[20])
	assert.Equal(t, byte(0x04), b[23])
	assert.Equal(t, byte(0x1A), b[24])
	assert.Equal(t, byte(0xE1), b[25])
}

func TestNodeInfoUnmarshalBinary(t *testing.T) {
	t.Parallel()

	data := make([]byte, 26)
	data[0] = 0xAB
	data[1] = 0xCD
	data[20] = 10
	data[21] = 11
	data[22] = 12
	data[23] = 13
	data[24] = 0x1A
	data[25] = 0xE1

	var ni NodeInfo

	err := ni.UnmarshalBinary(data)
	require.NoError(t, err)
	assert.Equal(t, byte(0xAB), ni.ID[0])
	assert.Equal(t, byte(0xCD), ni.ID[1])
	assert.Equal(t, "10.11.12.13", ni.Addr.IP.String())
	assert.Equal(t, 6881, ni.Addr.Port)
}

func TestNodeInfoMarshalUnmarshalRoundTrip(t *testing.T) {
	t.Parallel()

	ni := NodeInfo{
		ID:   testutil.MustParseID("ffeeddccbbaa9988776655443322110000112233"),
		Addr: NodeAddr{IP: net.IP{192, 168, 1, 1}, Port: 8888},
	}
	b, err := ni.MarshalBinary()
	require.NoError(t, err)

	var decoded NodeInfo

	err = decoded.UnmarshalBinary(b)
	require.NoError(t, err)
	assert.Equal(t, ni.ID, decoded.ID)
	assert.Equal(t, ni.Addr.IP.String(), decoded.Addr.IP.String())
	assert.Equal(t, ni.Addr.Port, decoded.Addr.Port)
}

func TestCompactIPv4NodeInfoElemSize(t *testing.T) {
	t.Parallel()

	ci := CompactIPv4NodeInfo{}
	assert.Equal(t, 26, ci.ElemSize())
}

func TestCompactIPv4NodeInfoMarshalUnmarshalBinary(t *testing.T) {
	t.Parallel()

	ni := []NodeInfo{
		{
			ID:   testutil.MustParseID("0102030405060708090a0b0c0d0e0f1011121314"),
			Addr: NodeAddr{IP: net.IP{1, 2, 3, 4}, Port: 6881},
		},
		{
			ID:   testutil.MustParseID("ffffffffffffffffffffffffffffffffffffffff"),
			Addr: NodeAddr{IP: net.IP{5, 6, 7, 8}, Port: 3334},
		},
	}
	ci := CompactIPv4NodeInfo(ni)
	b, err := ci.MarshalBinary()
	require.NoError(t, err)
	assert.Len(t, b, 52)

	var decoded CompactIPv4NodeInfo

	err = decoded.UnmarshalBinary(b)
	require.NoError(t, err)
	assert.Len(t, decoded, 2)
	assert.Equal(t, ni[0].ID, decoded[0].ID)
	assert.Equal(t, ni[0].Addr.IP.String(), decoded[0].Addr.IP.String())
	assert.Equal(t, ni[0].Addr.Port, decoded[0].Addr.Port)
}
