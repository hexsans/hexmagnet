package ktable

import (
	"net/netip"
	"time"

	"go.uber.org/fx"
)

type Params struct {
	fx.In
	NodeID ID `name:"dht_node_id"`
}

type Result struct {
	fx.Out
	Table Table
}

const (
	nodesK  = 80
	hashesK = 80
)

func New(p Params) Result {
	rm := &reverseMap{addrs: make(map[string]*infoForAddr)}
	nodes := nodeKeyspace{
		keyspace: newKeyspace[netip.AddrPort, NodeOption, Node, *node](
			p.NodeID,
			nodesK,
			func(id ID, addr netip.AddrPort) *node {
				return &node{
					nodeBase: nodeBase{
						id:   id,
						addr: addr,
					},
					discoveredAt: time.Now(),
					reverseMap:   rm,
				}
			},
		),
	}
	hashes := hashKeyspace{
		keyspace: newKeyspace[[]HashPeer, HashOption, Hash, *hash](
			p.NodeID,
			hashesK,
			func(id ID, peers []HashPeer) *hash {
				peersMap := make(map[string]HashPeer, len(peers))
				for _, p := range peers {
					peersMap[p.Addr.Addr().String()] = p
					rm.putAddrHashes(p.Addr.Addr(), id)
				}

				return &hash{
					id:           id,
					peers:        peersMap,
					discoveredAt: time.Now(),
					reverseMap:   rm,
				}
			},
		),
	}

	return Result{
		Table: &table{
			origin:  p.NodeID,
			nodesK:  nodesK,
			hashesK: hashesK,
			nodes:   nodes,
			hashes:  hashes,
			addrs:   rm,
		},
	}
}
