package shard

import (
	"os"
	"path/filepath"
	"sort"

	"gitlab.com/NebulousLabs/Sia/modules"
	"gitlab.com/NebulousLabs/Sia/persist"
	"gitlab.com/NebulousLabs/Sia/types"
)

// PersistData contains the data that a Relay loads on startup.
type PersistData struct {
	Height     types.BlockHeight
	Hosts      map[string][]byte
	LastChange modules.ConsensusChangeID
}

// A Persister can save and load PersistData.
type Persister interface {
	Save(PersistData) error
	Load(*PersistData) error
}

func (r *Relay) save() error {
	return r.persist.Save(PersistData{
		Height:     r.height,
		Hosts:      r.hosts,
		LastChange: r.lastChange,
	})
}

func (r *Relay) load() error {
	var data PersistData
	if err := r.persist.Load(&data); err != nil && !os.IsNotExist(err) {
		return err
	}
	if data.Hosts == nil {
		data.Hosts = make(map[string][]byte)
	}
	r.height = data.Height
	r.hosts = data.Hosts
	r.lastChange = data.LastChange
	for pk := range r.hosts {
		r.hostKeys = append(r.hostKeys, pk)
	}
	sort.Strings(r.hostKeys)
	return nil
}

var meta = persist.Metadata{
	Header:  "shard",
	Version: "0.1.0",
}

// JSONPersist implements Persister using a JSON file stored on disk.
type JSONPersist struct {
	path string
}

// Save implements Persister.
func (p JSONPersist) Save(data PersistData) error {
	return persist.SaveJSON(meta, data, p.path)
}

// Load implements Persister.
func (p JSONPersist) Load(data *PersistData) error {
	return persist.LoadJSON(meta, data, p.path)
}

// NewJSONPersist returns a new JSONPersist that writes to the specified
// directory.
func NewJSONPersist(dir string) JSONPersist {
	return JSONPersist{filepath.Join(dir, "persist.json")}
}
