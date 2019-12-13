package shard

import (
	"os"
	"sort"
	"strings"
	"sync"

	"gitlab.com/NebulousLabs/Sia/modules"
	"gitlab.com/NebulousLabs/Sia/types"
)

// A ConsensusSet provides updates to the Sia blockchain.
type ConsensusSet interface {
	ConsensusSetSubscribe(modules.ConsensusSetSubscriber, modules.ConsensusChangeID, <-chan struct{}) error
	Synced() bool
}

// A Relay watches the Sia blockchain for new host announcements and makes them
// available for queries.
type Relay struct {
	height     types.BlockHeight
	hosts      map[string][]byte // pubkey -> announcement
	hostKeys   []string          // sorted
	lastChange modules.ConsensusChangeID
	queuedSave bool
	cs         ConsensusSet
	persist    Persister
	mu         sync.Mutex
}

// Synced returns whether the Relay's blockchain is synced with the Sia network.
func (r *Relay) Synced() bool {
	return r.cs.Synced()
}

// Height returns the current blockchain height.
func (r *Relay) Height() types.BlockHeight {
	r.mu.Lock()
	height := r.height
	r.mu.Unlock()
	return height
}

// Host looks up a host using the given prefix of the host's public key. If more
// than one host shares the prefix, it returns false. If no host is found, it
// returns an empty string.
func (r *Relay) Host(prefix string) (pk string, unique bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.hosts[prefix]; ok {
		return prefix, true
	}
	i := sort.Search(len(r.hostKeys), func(i int) bool {
		hk := r.hostKeys[i]
		if len(prefix) > len(hk) {
			return hk[:len(prefix)] >= prefix
		}
		return hk[:len(prefix)] >= prefix
	})
	if i == len(r.hostKeys) || !strings.HasPrefix(r.hostKeys[i], prefix) {
		return "", false
	}
	pk = r.hostKeys[i]
	unique = i+1 == len(r.hostKeys) || !strings.HasPrefix(r.hostKeys[i+1], prefix)
	return
}

// HostAnnouncement returns the raw bytes of the announcement recorded in the
// Sia blockchain for the given host public key, or false if the host is not
// found.
func (r *Relay) HostAnnouncement(pubkey string) ([]byte, bool) {
	r.mu.Lock()
	ann, ok := r.hosts[pubkey]
	r.mu.Unlock()
	return ann, ok
}

// NewRelay initializes a Relay using the provided ConsensusSet and Persister.
func NewRelay(cs ConsensusSet, p Persister) (*Relay, error) {
	r := &Relay{
		hosts:   make(map[string][]byte),
		cs:      cs,
		persist: p,
	}
	if err := r.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// subscribe to consensus
	if err := cs.ConsensusSetSubscribe(r, r.lastChange, nil); err != nil {
		r.lastChange = modules.ConsensusChangeBeginning
		if err := cs.ConsensusSetSubscribe(r, r.lastChange, nil); err != nil {
			return nil, err
		}
	}
	return r, nil
}
