package ktable

import (
	"net/netip"
	"time"
)

const maxPeersPerHash = 30

type hashKeyspace struct {
	keyspace[[]HashPeer, HashOption, Hash, *hash]
}

type Hash interface {
	keyspaceItem
	Peers() []HashPeer
	Dropped() bool
}

type hash struct {
	id           ID
	peers        map[string]HashPeer
	discoveredAt time.Time
	// lastRequestedAt time.Time
	droppedReason error
	reverseMap    *reverseMap
}

type HashPeer struct {
	Addr netip.AddrPort
}

type HashOption interface {
	apply(*hash)
}

var _ keyspaceItemPrivate[[]HashPeer, HashOption, Hash] = (*hash)(nil)

func (h *hash) update(peers []HashPeer) {
	for _, p := range peers {
		h.peers[p.Addr.Addr().String()] = p
		h.reverseMap.putAddrHashes(p.Addr.Addr(), h.id)
	}

	excess := len(h.peers) - maxPeersPerHash
	for k, p := range h.peers {
		if excess <= 0 {
			break
		}

		delete(h.peers, k)
		h.reverseMap.dropHashForAddrs(h.id, p.Addr.Addr())

		excess--
	}
}

func (h *hash) apply(option HashOption) {
	option.apply(h)
}

func (h *hash) public() Hash {
	return h
}

func (h *hash) ID() ID {
	return h.id
}

func (h *hash) Peers() []HashPeer {
	peers := make([]HashPeer, 0, len(h.peers))
	for _, p := range h.peers {
		peers = append(peers, p)
	}

	return peers
}

func (h *hash) Dropped() bool {
	return h.droppedReason != nil
}
