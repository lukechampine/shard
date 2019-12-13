package shard

import (
	"io"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"lukechampine.com/us/hostdb"
)

type server struct {
	relay *Relay
}

func (s *server) handlerSynced(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	io.WriteString(w, strconv.FormatBool(s.relay.Synced()))
}

func (s *server) handlerHeight(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	io.WriteString(w, strconv.Itoa(int(s.relay.Height())))
}

func (s *server) handlerHost(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	pubkey, unique := s.relay.Host(hostdb.HostPublicKey(ps.ByName("prefix")))
	if pubkey == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	} else if !unique {
		http.Error(w, "ambiguous pubkey", http.StatusGone)
		return
	}
	ann, ok := s.relay.HostAnnouncement(pubkey)
	if !ok {
		// unlikely, but possible if an announcement is reverted after the call
		// to Host
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Write(ann)
}

// NewServer returns an http.Handler that serves the shard API using the
// provided Relay.
func NewServer(r *Relay) http.Handler {
	srv := &server{r}
	mux := httprouter.New()
	mux.GET("/synced", srv.handlerSynced)
	mux.GET("/height", srv.handlerHeight)
	mux.GET("/host/:prefix", srv.handlerHost)
	return mux
}
