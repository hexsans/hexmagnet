package ktable

import (
	"net/netip"
)

type reverseMap struct {
	addrs map[string]*infoForAddr
}

type infoForAddr struct {
	peerID ID
	hashes map[ID]struct{}
}

func newInfoForAddr(peerID ID, hashes ...ID) *infoForAddr {
	info := infoForAddr{
		peerID: peerID,
		hashes: make(map[ID]struct{}, len(hashes)),
	}
	info.addHashes(hashes...)

	return &info
}

func (i infoForAddr) addHashes(hashes ...ID) {
	for _, h := range hashes {
		i.hashes[h] = struct{}{}
	}
}

func (i infoForAddr) dropHashes(hashes ...ID) {
	for _, h := range hashes {
		delete(i.hashes, h)
	}
}

func (m reverseMap) putAddrPeerID(addr netip.Addr, id ID) {
	str := addr.String()
	if _, ok := m.addrs[str]; ok {
		m.addrs[str].peerID = id
	} else {
		m.addrs[str] = newInfoForAddr(id)
	}
}

func (m reverseMap) putAddrHashes(addr netip.Addr, hashes ...ID) {
	str := addr.String()
	if _, ok := m.addrs[str]; ok {
		m.addrs[str].addHashes(hashes...)
	} else {
		m.addrs[str] = newInfoForAddr(ID{}, hashes...)
	}
}

func (m reverseMap) dropAddr(addr netip.Addr) bool {
	if _, ok := m.addrs[addr.String()]; ok {
		delete(m.addrs, addr.String())
		return true
	}

	return false
}

func (m reverseMap) dropHashForAddrs(hash ID, addrs ...netip.Addr) {
	for _, addr := range addrs {
		addrStr := addr.String()
		if info, ok := m.addrs[addrStr]; ok {
			info.dropHashes(hash)

			if len(info.hashes) == 0 && info.peerID.IsZero() {
				delete(m.addrs, addrStr)
			}
		}
	}
}
