package ktable

import (
	"net/netip"

	"github.com/hexsans/hexmagnet/internal/protocol/dht/ktable/btree"
)

type Command interface {
	exec(*table)
}

type CommandReturn[T any] interface {
	Command
	Query[T]
}

var _ CommandReturn[btree.PutResult] = PutNode{}

type PutNode struct {
	ID      ID
	Addr    netip.AddrPort
	Options []NodeOption
}

func (c PutNode) execReturn(t *table) btree.PutResult {
	if !c.Addr.IsValid() {
		return btree.PutRejected
	}

	return t.nodes.put(c.ID, c.Addr, c.Options...)
}

func (c PutNode) exec(t *table) {
	c.execReturn(t)
}

var _ CommandReturn[bool] = DropNode{}

type DropNode struct {
	ID     ID
	Reason error
}

func (c DropNode) execReturn(t *table) bool {
	return t.nodes.drop(c.ID, c.Reason)
}

func (c DropNode) exec(t *table) {
	c.execReturn(t)
}

var _ CommandReturn[btree.PutResult] = PutHash{}

type PutHash struct {
	ID      ID
	Peers   []HashPeer
	Options []HashOption
}

func (c PutHash) execReturn(t *table) btree.PutResult {
	return t.hashes.put(c.ID, c.Peers, c.Options...)
}

func (c PutHash) exec(t *table) {
	c.execReturn(t)
}
