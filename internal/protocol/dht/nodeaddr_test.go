package dht

import (
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalNodeAddr(t *testing.T) {
	t.Parallel()

	var na NodeAddr
	require.NoError(t, na.UnmarshalBinary([]byte("\x01\x02\x03\x04\x05\x06")))
	assert.Equal(t, "1.2.3.4", na.IP.String())
}

func TestNewNodeAddrFromAddrPort(t *testing.T) {
	t.Parallel()

	ap := netip.MustParseAddrPort("1.2.3.4:6881")
	na := NewNodeAddrFromAddrPort(ap)
	assert.Equal(t, "1.2.3.4", na.IP.String())
	assert.Equal(t, 6881, na.Port)
}

func TestNodeAddrString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		na   NodeAddr
		want string
	}{
		{"ipv4", NodeAddr{IP: net.IP{1, 2, 3, 4}, Port: 6881}, "1.2.3.4:6881"},
		{"ipv6", NodeAddr{IP: net.ParseIP("::1"), Port: 3334}, "[::1]:3334"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.na.String())
		})
	}
}

func TestNodeAddrToAddrPort(t *testing.T) {
	t.Parallel()

	na := NodeAddr{IP: net.IP{1, 2, 3, 4}, Port: 6881}
	ap := na.ToAddrPort()
	assert.Equal(t, "1.2.3.4:6881", ap.String())
	assert.Equal(t, uint16(6881), ap.Port())
}

func TestNodeAddrMarshalBinary(t *testing.T) {
	t.Parallel()

	na := NodeAddr{IP: net.IP{1, 2, 3, 4}, Port: 6881}
	b, err := na.MarshalBinary()
	require.NoError(t, err)
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04, 0x1A, 0xE1}, b)
}

func TestNodeAddrUnmarshalBencode(t *testing.T) {
	t.Parallel()

	bencoded := "6:\x01\x02\x03\x04\x1a\xe1"

	var na NodeAddr

	err := na.UnmarshalBencode([]byte(bencoded))
	require.NoError(t, err)
	assert.Equal(t, "1.2.3.4", na.IP.String())
	assert.Equal(t, 6881, na.Port)
}

func TestNodeAddrMarshalUnmarshalRoundTrip(t *testing.T) {
	t.Parallel()

	original := NodeAddr{IP: net.IP{10, 20, 30, 40}, Port: 9999}
	b, err := original.MarshalBinary()
	require.NoError(t, err)

	var decoded NodeAddr

	err = decoded.UnmarshalBinary(b)
	require.NoError(t, err)
	assert.True(t, original.Equal(decoded))
}

func TestNodeAddrUDP(t *testing.T) {
	t.Parallel()

	na := NodeAddr{IP: net.IP{1, 2, 3, 4}, Port: 6881}
	udp := na.UDP()
	assert.Equal(t, "1.2.3.4", udp.IP.String())
	assert.Equal(t, 6881, udp.Port)
}

func TestNodeAddrFromUDPAddr(t *testing.T) {
	t.Parallel()

	ua := &net.UDPAddr{IP: net.ParseIP("5.6.7.8"), Port: 1234}

	var na NodeAddr
	na.FromUDPAddr(ua)
	assert.Equal(t, "5.6.7.8", na.IP.String())
	assert.Equal(t, 1234, na.Port)
}

func TestNodeAddrEqual(t *testing.T) {
	t.Parallel()

	a := NodeAddr{IP: net.IP{1, 2, 3, 4}, Port: 6881}
	b := NodeAddr{IP: net.IP{1, 2, 3, 4}, Port: 6881}
	c := NodeAddr{IP: net.IP{1, 2, 3, 5}, Port: 6881}
	d := NodeAddr{IP: net.IP{1, 2, 3, 4}, Port: 6882}

	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(c))
	assert.False(t, a.Equal(d))
}

func TestNodeAddrZeroPortString(t *testing.T) {
	t.Parallel()

	na := NodeAddr{IP: net.ParseIP("0.0.0.0"), Port: 0}
	assert.Equal(t, "0.0.0.0:0", na.String())
}
