package ktable

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/hexsans/hexmagnet/internal/protocol"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable/btree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func testID(prefix byte) protocol.ID {
	var raw [20]byte

	raw[0] = prefix

	return protocol.ID(raw)
}

func newTestTable(t *testing.T) Table {
	t.Helper()

	origin := testID(0)
	result := New(Params{
		NodeID: origin,
	})

	return result.Table
}

func TestTable_Origin(t *testing.T) {
	t.Parallel()

	origin := testID(0)
	tbl := New(Params{NodeID: origin}).Table

	assert.Equal(t, origin, tbl.Origin())
}

func TestTable_PutNode_InvalidAddr(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)
	nodeID := testID(1)

	result := tbl.PutNode(nodeID, netip.AddrPort{}, NodeResponded())
	assert.Equal(t, btree.PutRejected, result)
}

func TestTable_PutNode_Accepted(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)
	nodeID := testID(1)
	addr := netip.MustParseAddrPort("1.2.3.4:6881")

	result := tbl.PutNode(nodeID, addr)
	assert.Equal(t, btree.PutAccepted, result)
}

func TestTable_PutNode_AlreadyExists(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)
	nodeID := testID(1)
	addr := netip.MustParseAddrPort("1.2.3.4:6881")

	_ = tbl.PutNode(nodeID, addr)
	result := tbl.PutNode(nodeID, addr)
	assert.Equal(t, btree.PutAlreadyExists, result)
}

func TestTable_DropNode(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)
	nodeID := testID(1)
	addr := netip.MustParseAddrPort("1.2.3.4:6881")

	_ = tbl.PutNode(nodeID, addr)
	dropped := tbl.DropNode(nodeID, nil)
	assert.True(t, dropped)

	dropped = tbl.DropNode(nodeID, nil)
	assert.False(t, dropped)
}

func TestTable_GetClosestNodes_WithNode(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)

	nodes := []protocol.ID{testID(3), testID(4), testID(5), testID(6)}
	for _, id := range nodes {
		tbl.PutNode(id, netip.MustParseAddrPort("1.2.3.4:6881"))
	}

	closest := tbl.GetClosestNodes(testID(3))
	require.NotEmpty(t, closest)
	assert.Equal(t, testID(3), closest[0].ID())
}

func TestTable_GetClosestNodes_WithoutNode(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)

	for i := range 8 {
		tbl.PutNode(testID(byte(10+i)), netip.MustParseAddrPort("1.2.3.4:6881"))
	}

	closest := tbl.GetClosestNodes(testID(2))
	require.NotEmpty(t, closest)
	assert.LessOrEqual(t, len(closest), 8)
}

func TestTable_GetOldestNodes(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)

	for i := range 5 {
		tbl.PutNode(testID(byte(20+i)), netip.MustParseAddrPort("1.2.3.4:6881"), NodeResponded())
	}

	oldest := tbl.GetOldestNodes(time.Now().Add(time.Minute), 10)
	assert.Len(t, oldest, 5)

	oldest = tbl.GetOldestNodes(time.Now().Add(-time.Hour), 10)
	assert.Empty(t, oldest)
}

func TestTable_PutHash(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)
	hashID := testID(0x10)
	peer := HashPeer{Addr: netip.MustParseAddrPort("5.6.7.8:6881")}

	pres := tbl.PutHash(hashID, []HashPeer{peer})
	assert.Equal(t, btree.PutAccepted, pres)

	gres := tbl.GetHashOrClosestNodes(hashID)
	assert.True(t, gres.Found)
	assert.NotNil(t, gres.Hash)
	assert.Equal(t, hashID, gres.Hash.ID())
	require.Len(t, gres.Hash.Peers(), 1)
	assert.Equal(t, peer.Addr, gres.Hash.Peers()[0].Addr)
}

func TestTable_GetHashOrClosestNodes_Miss(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)

	for i := range 4 {
		tbl.PutNode(testID(byte(30+i)), netip.MustParseAddrPort("1.2.3.4:6881"))
	}

	gres := tbl.GetHashOrClosestNodes(testID(0x99))
	assert.False(t, gres.Found)
	assert.Nil(t, gres.Hash)
	assert.NotEmpty(t, gres.ClosestNodes)
}

func TestTable_FilterKnownAddrs(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)

	knownAddr := netip.MustParseAddrPort("10.0.0.1:6881")
	tbl.PutNode(testID(0x20), knownAddr)

	tbl.PutNode(testID(0x20), knownAddr)

	unknownAddr := netip.MustParseAddr("10.0.0.2")
	knownIP := netip.MustParseAddr("10.0.0.1")

	filtered := tbl.FilterKnownAddrs([]netip.Addr{knownIP, unknownAddr})
	require.Len(t, filtered, 1)
	assert.Equal(t, unknownAddr, filtered[0])
}

func TestTable_SampleHashesAndNodes(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)

	for i := range 5 {
		tbl.PutNode(testID(byte(0x30+i)), netip.MustParseAddrPort("1.2.3.4:6881"))
	}

	for i := range 3 {
		tbl.PutHash(testID(byte(0x40+i)), []HashPeer{
			{Addr: netip.MustParseAddrPort("5.6.7.8:6881")},
		})
	}

	sample := tbl.SampleHashesAndNodes()
	assert.Len(t, sample.Hashes, 3)
	assert.NotEmpty(t, sample.Nodes)
	assert.Equal(t, 3, sample.TotalHashes)
}

func TestTable_BatchCommand(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)

	tbl.BatchCommand(
		PutNode{
			ID:   testID(0x50),
			Addr: netip.MustParseAddrPort("10.0.0.1:6881"),
		},
		PutNode{
			ID:   testID(0x51),
			Addr: netip.MustParseAddrPort("10.0.0.2:6881"),
		},
	)

	closest := tbl.GetClosestNodes(testID(0x50))
	require.Len(t, closest, 1)
	assert.Equal(t, testID(0x50), closest[0].ID())
}

func TestTable_GetNodesForSampleInfoHashes(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)

	for i := range 10 {
		tbl.PutNode(testID(byte(0x60+i)), netip.MustParseAddrPort("1.2.3.4:6881"))
	}

	candidates := tbl.GetNodesForSampleInfoHashes(5)
	assert.Len(t, candidates, 5)
}

func TestTable_GetNodesForSampleInfoHashes_NoCandidates(t *testing.T) {
	t.Parallel()

	tbl := newTestTable(t)

	nodes := tbl.GetNodesForSampleInfoHashes(5)
	assert.Empty(t, nodes)
}

func TestTable_FxProvide(t *testing.T) {
	t.Parallel()

	origin := testID(0)
	app := fx.New(
		fx.Provide(
			fx.Annotate(
				func() ID { return origin },
				fx.ResultTags(`name:"dht_node_id"`),
			),
			New,
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := app.Start(ctx)
	require.NoError(t, err)

	err = app.Stop(ctx)
	require.NoError(t, err)
}
