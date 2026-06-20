package ktable

import (
	"net/netip"
	"testing"

	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestReverseMap_PutAddrPeerID(t *testing.T) {
	t.Parallel()

	rm := reverseMap{addrs: make(map[string]*infoForAddr)}
	addr := netip.MustParseAddr("1.2.3.4")
	id := testutil.MustParseID("1111111111111111111111111111111111111111")

	rm.putAddrPeerID(addr, id)
	info, ok := rm.addrs[addr.String()]
	assert.True(t, ok)
	assert.Equal(t, id, info.peerID)
}

func TestReverseMap_PutAddrPeerID_Overwrite(t *testing.T) {
	t.Parallel()

	rm := reverseMap{addrs: make(map[string]*infoForAddr)}
	addr := netip.MustParseAddr("1.2.3.4")
	id1 := testutil.MustParseID("1111111111111111111111111111111111111111")
	id2 := testutil.MustParseID("2222222222222222222222222222222222222222")

	rm.putAddrPeerID(addr, id1)
	rm.putAddrPeerID(addr, id2)

	info, ok := rm.addrs[addr.String()]
	assert.True(t, ok)
	assert.Equal(t, id2, info.peerID)
}

func TestReverseMap_PutAddrHashes(t *testing.T) {
	t.Parallel()

	rm := reverseMap{addrs: make(map[string]*infoForAddr)}
	addr := netip.MustParseAddr("1.2.3.4")
	hash1 := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	hash2 := testutil.MustParseID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	rm.putAddrHashes(addr, hash1, hash2)
	assert.Contains(t, rm.addrs[addr.String()].hashes, hash1)
	assert.Contains(t, rm.addrs[addr.String()].hashes, hash2)
}

func TestReverseMap_DropAddr(t *testing.T) {
	t.Parallel()

	rm := reverseMap{addrs: make(map[string]*infoForAddr)}
	addr := netip.MustParseAddr("1.2.3.4")
	rm.putAddrPeerID(addr, testutil.MustParseID("1111111111111111111111111111111111111111"))

	dropped := rm.dropAddr(addr)
	assert.True(t, dropped)

	dropped = rm.dropAddr(addr)
	assert.False(t, dropped)
}

func TestReverseMap_DropAddr_WithHashes(t *testing.T) {
	t.Parallel()

	rm := reverseMap{addrs: make(map[string]*infoForAddr)}
	addr := netip.MustParseAddr("1.2.3.4")
	hash := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	rm.putAddrHashes(addr, hash)
	assert.Len(t, rm.addrs[addr.String()].hashes, 1)

	rm.dropHashForAddrs(hash, addr)
	assert.NotContains(t, rm.addrs, addr.String())
}

func TestReverseMap_DropHashForAddrs_PreservesPeerMapping(t *testing.T) {
	t.Parallel()

	rm := reverseMap{addrs: make(map[string]*infoForAddr)}
	addr := netip.MustParseAddr("1.2.3.4")
	hash := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	peerID := testutil.MustParseID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	rm.putAddrHashes(addr, hash)
	rm.putAddrPeerID(addr, peerID)

	rm.dropHashForAddrs(hash, addr)

	info := rm.addrs[addr.String()]
	assert.NotNil(t, info)
	assert.Equal(t, peerID, info.peerID)
	assert.Empty(t, info.hashes)
}

func TestInfoForAddr_AddAndDropHashes(t *testing.T) {
	t.Parallel()

	info := newInfoForAddr(protocol.ID{})
	hash := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	info.addHashes(hash)
	assert.Contains(t, info.hashes, hash)

	info.dropHashes(hash)
	assert.Empty(t, info.hashes)
}

func TestInfoForAddr_WithPeerID(t *testing.T) {
	t.Parallel()

	peerID := testutil.MustParseID("1111111111111111111111111111111111111111")
	info := newInfoForAddr(peerID)
	assert.Equal(t, peerID, info.peerID)
	assert.Empty(t, info.hashes)
}
