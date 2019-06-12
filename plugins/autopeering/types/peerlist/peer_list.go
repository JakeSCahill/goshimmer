package peerlist

import (
	"sort"

	"github.com/iotaledger/goshimmer/plugins/autopeering/types/peer"
)

type PeerList []*peer.Peer

func (this PeerList) Clone() PeerList {
	result := make(PeerList, len(this))
	for i, entry := range this {
		result[i] = entry
	}

	return result
}

func (this PeerList) Filter(predicate func(p *peer.Peer) bool) PeerList {
	peerList := make(PeerList, len(this))

	counter := 0
	for _, peer := range this {
		if predicate(peer) {
			peerList[counter] = peer
			counter++
		}
	}

	return peerList[:counter]
}

// Sorts the PeerRegister by their distance to an anchor.
func (this PeerList) Sort(distance func(p *peer.Peer) uint64) PeerList {
	sort.Slice(this, func(i, j int) bool {
		return distance(this[i]) < distance(this[j])
	})

	return this
}
