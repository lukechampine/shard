package shard

import (
	"os"
	"path/filepath"
	"sort"

	"gitlab.com/NebulousLabs/Sia/modules"
	"gitlab.com/NebulousLabs/Sia/persist"
	"gitlab.com/NebulousLabs/Sia/types"
)

type PersistData struct {
	Height     types.BlockHeight
	Hosts      map[string][]byte
	LastChange modules.ConsensusChangeID
}

type Persister interface {
	Save(PersistData) error
	Load(*PersistData) error
}

func (s *SHARD) save() error {
	return s.persist.Save(PersistData{
		Height:     s.height,
		Hosts:      s.hosts,
		LastChange: s.lastChange,
	})
}

func (s *SHARD) load() error {
	var data PersistData
	if err := s.persist.Load(&data); err != nil && !os.IsNotExist(err) {
		return err
	}
	if data.Hosts == nil {
		data.Hosts = make(map[string][]byte)
	}
	s.height = data.Height
	s.hosts = data.Hosts
	s.lastChange = data.LastChange
	for pk := range s.hosts {
		s.hostKeys = append(s.hostKeys, pk)
	}
	sort.Strings(s.hostKeys)
	return nil
}

var meta = persist.Metadata{
	Header:  "shard",
	Version: "0.1.0",
}

type JSONPersist struct {
	path string
}

func (p JSONPersist) Save(data PersistData) error {
	return persist.SaveJSON(meta, data, p.path)
}

func (p JSONPersist) Load(data *PersistData) error {
	return persist.LoadJSON(meta, data, p.path)
}

func NewJSONPersist(dir string) JSONPersist {
	return JSONPersist{filepath.Join(dir, "persist.json")}
}
