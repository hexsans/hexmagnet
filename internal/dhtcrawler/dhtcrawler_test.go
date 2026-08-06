package dhtcrawler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/netip"
	"testing"
	"time"

	bloomv3 "github.com/bits-and-blooms/bloom/v3"
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/server"
	"github.com/hexsans/hexmagnet/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSeenPeersBloom(t *testing.T) {
	t.Parallel()

	b := newSeenPeersBloom()
	require.NotNil(t, b)
	assert.NotNil(t, b.filter)
	assert.Equal(t, b.expectedCap, b.filter.Cap())
}

func TestSeenPeersBloom_TestAndAdd(t *testing.T) {
	t.Parallel()

	b := newSeenPeersBloom()
	addr := netip.MustParseAddr("1.2.3.4")
	addr2 := netip.MustParseAddr("5.6.7.8")

	assert.False(t, b.TestAndAdd(addr), "first add should return false")
	assert.True(t, b.TestAndAdd(addr), "second add should return true")
	assert.False(t, b.TestAndAdd(addr2), "different addr should return false")
}

func TestSeenPeersBloom_GobEncodeDecode(t *testing.T) {
	t.Parallel()

	b := newSeenPeersBloom()
	b.TestAndAdd(netip.MustParseAddr("1.2.3.4"))
	b.TestAndAdd(netip.MustParseAddr("5.6.7.8"))

	data, err := b.GobEncode()
	require.NoError(t, err)
	assert.Greater(t, len(data), 8)

	b2 := newSeenPeersBloom()
	err = b2.GobDecode(data)
	require.NoError(t, err)

	assert.True(t, b2.TestAndAdd(netip.MustParseAddr("1.2.3.4")), "should still return true")
	assert.True(t, b2.TestAndAdd(netip.MustParseAddr("5.6.7.8")), "should still return true")
	assert.False(t, b2.TestAndAdd(netip.MustParseAddr("9.9.9.9")), "new addr should return false")
}

func TestSeenPeersBloom_GobDecode_TooShort(t *testing.T) {
	t.Parallel()

	b := newSeenPeersBloom()
	err := b.GobDecode([]byte{1, 2, 3})
	assert.ErrorContains(t, err, "bloom data too short")
}

func TestSeenPeersBloom_GobDecode_CapacityMismatch(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	bf := bloomv3.NewWithEstimates(100, 0.01)
	_, writeErr := bf.WriteTo(&buf)
	require.NoError(t, writeErr)

	b := newSeenPeersBloom()
	err := b.GobDecode(buf.Bytes())
	assert.ErrorContains(t, err, "bloom capacity mismatch")
}

func TestSeenPeersBloom_WriteTo(t *testing.T) {
	t.Parallel()

	b := newSeenPeersBloom()
	b.TestAndAdd(netip.MustParseAddr("1.2.3.4"))

	var buf bytes.Buffer

	n, err := b.WriteTo(&buf)
	require.NoError(t, err)
	assert.Positive(t, n)
	assert.Positive(t, buf.Len())
}

func TestNewRuntime(t *testing.T) {
	t.Parallel()

	r := NewRuntime()
	require.NotNil(t, r)
	assert.NotNil(t, r.Active)
	assert.NotNil(t, r.PeersDiscovered)
	assert.NotNil(t, r.PeersConnected)
	assert.NotNil(t, r.StartedAt)
	assert.NotNil(t, r.SeenConnectedPeers)
	assert.NotNil(t, r.SeenDiscoveredPeers)
}

func TestRuntime_Uptime_NoStartTime(t *testing.T) {
	t.Parallel()

	r := NewRuntime()
	r.SetUptimeOffset(5 * time.Minute)
	uptime := r.Uptime()
	assert.Equal(t, 5*time.Minute, uptime)
}

func TestRuntime_Uptime_WithStartTime(t *testing.T) {
	t.Parallel()

	r := NewRuntime()
	r.SetUptimeOffset(10 * time.Second)
	r.StartedAt.Set(time.Now().Add(-30 * time.Second))

	uptime := r.Uptime()
	assert.GreaterOrEqual(t, uptime, 40*time.Second)
	assert.Less(t, uptime, 50*time.Second)
}

func TestRuntime_Uptime_ZeroValue(t *testing.T) {
	t.Parallel()

	r := NewRuntime()
	uptime := r.Uptime()
	assert.Equal(t, time.Duration(0), uptime)
}

func TestRuntime_SetUptimeOffset(t *testing.T) {
	t.Parallel()

	r := NewRuntime()
	r.SetUptimeOffset(10 * time.Minute)
	assert.Equal(t, 10*time.Minute, r.Uptime())
}

func TestRuntime_SetActivity(t *testing.T) {
	t.Parallel()

	r := NewRuntime()
	entries := []ActivityEntry{
		{ID: "1", Type: "test", Message: "hello", Time: time.Now()},
	}
	r.SetActivity(entries)
	assert.Equal(t, entries, r.RecentActivity())
}

func TestRuntime_PushActivity(t *testing.T) {
	t.Parallel()

	r := NewRuntime()
	r.PushActivity("info", "first")
	r.PushActivity("warn", "second")

	recent := r.RecentActivity()
	require.Len(t, recent, 2)
	assert.Equal(t, "warn", recent[0].Type)
	assert.Equal(t, "second", recent[0].Message)
	assert.Equal(t, "info", recent[1].Type)
	assert.Equal(t, "first", recent[1].Message)
}

func TestRuntime_PushActivity_Truncate(t *testing.T) {
	t.Parallel()

	r := NewRuntime()
	for range 60 {
		r.PushActivity("test", "msg")
	}

	recent := r.RecentActivity()
	assert.Len(t, recent, maxActivityEntries)
}

func TestRuntime_RecentActivity_ReturnsCopy(t *testing.T) {
	t.Parallel()

	r := NewRuntime()
	r.PushActivity("test", "original")

	recent := r.RecentActivity()
	recent[0].Message = "modified"

	actual := r.RecentActivity()
	assert.Equal(t, "original", actual[0].Message)
}

func TestIgnoreHashes_TestAndAdd(t *testing.T) {
	t.Parallel()

	i := &ignoreHashes{
		bloom: bloomv3.NewWithEstimates(1000, 0.01),
	}

	id1 := testutil.MustParseID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	id2 := testutil.MustParseID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	assert.False(t, i.testAndAdd(id1))
	assert.True(t, i.testAndAdd(id1))
	assert.False(t, i.testAndAdd(id2))
}

func TestNewHealthCheck_Active(t *testing.T) {
	t.Parallel()

	active := &concurrency.AtomicValue[bool]{}
	active.Set(true)

	lastResponses := &concurrency.AtomicValue[server.LastResponses]{}

	check := NewHealthCheck(active, lastResponses)
	assert.Equal(t, "dht", check.Name)
	assert.Equal(t, time.Second, check.Timeout)
	assert.True(t, check.IsActive())
}

func TestNewHealthCheck_Inactive(t *testing.T) {
	t.Parallel()

	active := &concurrency.AtomicValue[bool]{}
	active.Set(false)

	check := NewHealthCheck(active, &concurrency.AtomicValue[server.LastResponses]{})
	assert.False(t, check.IsActive())
}

func TestNewHealthCheck_Check_NoStartTime(t *testing.T) {
	t.Parallel()

	active := &concurrency.AtomicValue[bool]{}
	lastResponses := &concurrency.AtomicValue[server.LastResponses]{}

	check := NewHealthCheck(active, lastResponses)
	err := check.Check(context.Background())
	assert.NoError(t, err)
}

func TestNewHealthCheck_Check_NoSuccessWithin30s(t *testing.T) {
	t.Parallel()

	active := &concurrency.AtomicValue[bool]{}
	lastResponses := &concurrency.AtomicValue[server.LastResponses]{}
	lastResponses.Set(server.LastResponses{
		StartTime: time.Now().Add(-45 * time.Second),
	})

	check := NewHealthCheck(active, lastResponses)
	err := check.Check(context.Background())
	assert.ErrorContains(t, err, "no response within 30 seconds")
}

func TestNewHealthCheck_Check_WithinGracePeriod(t *testing.T) {
	t.Parallel()

	active := &concurrency.AtomicValue[bool]{}
	lastResponses := &concurrency.AtomicValue[server.LastResponses]{}
	lastResponses.Set(server.LastResponses{
		StartTime: time.Now().Add(-10 * time.Second),
	})

	check := NewHealthCheck(active, lastResponses)
	err := check.Check(context.Background())
	assert.NoError(t, err)
}

func TestNewHealthCheck_Check_RecentSuccess(t *testing.T) {
	t.Parallel()

	active := &concurrency.AtomicValue[bool]{}
	lastResponses := &concurrency.AtomicValue[server.LastResponses]{}
	lastResponses.Set(server.LastResponses{
		StartTime:   time.Now().Add(-1 * time.Hour),
		LastSuccess: time.Now().Add(-10 * time.Second),
	})

	check := NewHealthCheck(active, lastResponses)
	err := check.Check(context.Background())
	assert.NoError(t, err)
}

func TestNewHealthCheck_Check_StaleSuccess(t *testing.T) {
	t.Parallel()

	active := &concurrency.AtomicValue[bool]{}
	lastResponses := &concurrency.AtomicValue[server.LastResponses]{}
	lastResponses.Set(server.LastResponses{
		StartTime:   time.Now().Add(-1 * time.Hour),
		LastSuccess: time.Now().Add(-2 * time.Minute),
	})

	check := NewHealthCheck(active, lastResponses)
	err := check.Check(context.Background())
	assert.ErrorContains(t, err, "no successful responses within last minute")
}

func TestNewDiscoveredNodes(t *testing.T) {
	t.Parallel()

	cfg := Config{
		BootstrapNodes:               defaultBootstrapNodes,
		ReseedBootstrapNodesInterval: time.Minute,
		RescrapeThreshold:            2592000,
		HashDiscoverLimit:            10,
		EmbedTrackers:                []string{},
	}
	params := DiscoveredNodesParams{Config: cfg}
	result := NewDiscoveredNodes(params)
	require.NotNil(t, result.DiscoveredNodes)
}

func TestNewHealthCheckFunction(t *testing.T) {
	t.Parallel()

	active := &concurrency.AtomicValue[bool]{}
	active.Set(true)

	lastResponses := &concurrency.AtomicValue[server.LastResponses]{}

	check := NewHealthCheck(active, lastResponses)
	assert.NotNil(t, check)
	assert.Equal(t, "dht", check.Name)
}

func TestRuntimeStateSerialization(t *testing.T) {
	t.Parallel()

	now := time.Now()
	state := RuntimeState{
		PeersConnected:  100,
		PeersDiscovered: 200,
		UptimeSeconds:   3600,
		Activity: []ActivityEntry{
			{ID: "1", Type: "info", Message: "started", Time: now},
		},
	}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	var decoded RuntimeState

	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, state.PeersConnected, decoded.PeersConnected)
	assert.Equal(t, state.PeersDiscovered, decoded.PeersDiscovered)
	assert.Equal(t, state.UptimeSeconds, decoded.UptimeSeconds)
	require.Len(t, decoded.Activity, 1)
	assert.Equal(t, "started", decoded.Activity[0].Message)
}
