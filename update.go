package shard

import (
	"sort"
	"time"

	"gitlab.com/NebulousLabs/Sia/crypto"
	"gitlab.com/NebulousLabs/Sia/encoding"
	"gitlab.com/NebulousLabs/Sia/modules"
	"gitlab.com/NebulousLabs/Sia/types"
)

type hostAnnouncement struct {
	modules.HostAnnouncement
	Signature crypto.Signature
}

func addHostAnnouncements(b types.Block, hosts map[string][]byte) {
	for _, t := range b.Transactions {
		for _, arb := range t.ArbitraryData {
			// decode announcement
			var ha hostAnnouncement
			if err := encoding.Unmarshal(arb, &ha); err != nil {
				continue
			} else if ha.Specifier != modules.PrefixHostAnnouncement {
				continue
			}

			// verify signature
			var pk crypto.PublicKey
			copy(pk[:], ha.PublicKey.Key)
			annHash := crypto.HashObject(ha.HostAnnouncement)
			if err := crypto.VerifyHash(annHash, pk, ha.Signature); err != nil {
				continue
			}
			// make a copy -- don't want to store pointers to consensus memory
			hosts[ha.PublicKey.String()] = append([]byte(nil), arb...)
		}
	}
}

// ProcessConsensusChange implements modules.ConsensusSetSubscriber.
func (r *Relay) ProcessConsensusChange(cc modules.ConsensusChange) {
	// find host announcements
	newhosts := make(map[string][]byte)
	for _, block := range cc.AppliedBlocks {
		addHostAnnouncements(block, newhosts)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// update height
	r.height += types.BlockHeight(len(cc.AppliedBlocks))
	r.height -= types.BlockHeight(len(cc.RevertedBlocks))
	if r.lastChange == modules.ConsensusChangeBeginning {
		r.height-- // genesis block is height 0
	}

	// add host announcements
	for pk, ann := range newhosts {
		if _, seen := r.hosts[pk]; !seen {
			r.hostKeys = append(r.hostKeys, pk)
		}
		r.hosts[pk] = ann
	}
	sort.Strings(r.hostKeys)

	// mark this set of blocks as processed
	r.lastChange = cc.ID

	// Queue a save in the near future. If there is already a queued save, do
	// nothing. This strategy ensures that we eventually save new hosts, but
	// avoids saving after every block.
	if len(newhosts) > 0 && !r.queuedSave {
		r.queuedSave = true
		time.AfterFunc(2*time.Minute, func() {
			r.mu.Lock()
			r.save()
			r.queuedSave = false
			r.mu.Unlock()
		})
	}
}
