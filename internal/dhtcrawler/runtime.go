package dhtcrawler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
	"sync"
	"time"

	bloomv3 "github.com/bits-and-blooms/bloom/v3"
	"github.com/hexsans/hexmagnet/internal/concurrency"
	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable"
)

type SeenPeersBloom struct {
	mu          sync.Mutex
	filter      *bloomv3.BloomFilter
	expectedCap uint
}

func newSeenPeersBloom() *SeenPeersBloom {
	f := bloomv3.NewWithEstimates(10_000_000, 0.001)

	return &SeenPeersBloom{
		filter:      f,
		expectedCap: f.Cap(),
	}
}

func (s *SeenPeersBloom) TestAndAdd(addr netip.Addr) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := addr.As16()

	return s.filter.TestAndAdd(key[:])
}

func (s *SeenPeersBloom) GobEncode() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var buf bytes.Buffer
	if _, err := s.filter.WriteTo(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *SeenPeersBloom) GobDecode(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("bloom data too short (%d bytes)", len(data))
	}

	persistedM := binary.BigEndian.Uint64(data[:8])
	if persistedM != uint64(s.expectedCap) {
		return fmt.Errorf("bloom capacity mismatch: persisted %d, expected %d", persistedM, s.expectedCap)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	bf := &bloomv3.BloomFilter{}
	if _, err := bf.ReadFrom(bytes.NewReader(data)); err != nil {
		return err
	}

	s.filter = bf

	return nil
}

func (s *SeenPeersBloom) WriteTo(w io.Writer) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.filter.WriteTo(w)
}

type ActivityEntry struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

type Runtime struct {
	Active          *concurrency.AtomicValue[bool]
	PeersDiscovered *concurrency.AtomicValue[uint64]
	PeersConnected  *concurrency.AtomicValue[uint64]
	uptimeOffset    time.Duration
	StartedAt       *concurrency.AtomicValue[time.Time]

	SeenConnectedPeers  *SeenPeersBloom
	SeenDiscoveredPeers *SeenPeersBloom

	NodesForPing             concurrency.BufferedConcurrentChannel[ktable.Node]
	NodesForFindNode         concurrency.BufferedConcurrentChannel[ktable.Node]
	NodesForSampleInfoHashes concurrency.BufferedConcurrentChannel[ktable.Node]

	mu       sync.Mutex
	activity []ActivityEntry
}

func NewRuntime() *Runtime {
	return &Runtime{
		Active:              &concurrency.AtomicValue[bool]{},
		PeersConnected:      &concurrency.AtomicValue[uint64]{},
		PeersDiscovered:     &concurrency.AtomicValue[uint64]{},
		StartedAt:           &concurrency.AtomicValue[time.Time]{},
		SeenConnectedPeers:  newSeenPeersBloom(),
		SeenDiscoveredPeers: newSeenPeersBloom(),
	}
}

func (r *Runtime) Uptime() time.Duration {
	startedAt := r.StartedAt.Get()
	if startedAt.IsZero() {
		return r.uptimeOffset
	}

	return r.uptimeOffset + time.Since(startedAt)
}

func (r *Runtime) SetUptimeOffset(d time.Duration) {
	r.uptimeOffset = d
}

const maxActivityEntries = 50

func (r *Runtime) SetActivity(entries []ActivityEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.activity = entries
}

func (r *Runtime) PushActivity(typ, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.activity = append([]ActivityEntry{{
		ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:    typ,
		Message: message,
		Time:    time.Now(),
	}}, r.activity...)
	if len(r.activity) > maxActivityEntries {
		r.activity = r.activity[:maxActivityEntries]
	}
}

func (r *Runtime) RecentActivity() []ActivityEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]ActivityEntry, len(r.activity))
	copy(result, r.activity)

	return result
}
