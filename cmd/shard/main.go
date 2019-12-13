package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"gitlab.com/NebulousLabs/Sia/build"
	"gitlab.com/NebulousLabs/Sia/modules/consensus"
	"gitlab.com/NebulousLabs/Sia/modules/gateway"
	"lukechampine.com/shard"
)

var (
	// to be supplied at build time
	githash   = "?"
	builddate = "?"
)

func main() {
	persistDir := flag.String("d", ".", "directory where server state is stored")
	rpcAddr := flag.String("r", ":9381", "host:port that the gateway listens on")
	apiAddr := flag.String("a", ":8080", "host:port that the API server listens on")
	flag.Parse()

	if len(flag.Args()) == 1 && flag.Arg(0) == "version" {
		log.SetFlags(0)
		log.Printf("shard v0.2.0\nCommit:     %s\nRelease:    %s\nGo version: %s %s/%s\nBuild Date: %s\n",
			githash, build.Release, runtime.Version(), runtime.GOOS, runtime.GOARCH, builddate)
		return
	} else if len(flag.Args()) != 0 {
		flag.Usage()
		return
	}

	g, err := gateway.New(*rpcAddr, true, filepath.Join(*persistDir, "gateway"))
	if err != nil {
		log.Fatal(err)
	}
	cs, errCh := consensus.New(g, true, filepath.Join(*persistDir, "consensus"))
	if err := <-errCh; err != nil {
		log.Fatal(err)
	}
	relay, err := shard.NewRelay(cs, shard.NewJSONPersist(*persistDir))
	if err != nil {
		log.Fatal(err)
	}

	srv := shard.NewServer(relay)
	log.Printf("Listening on %v...", *apiAddr)
	log.Fatal(http.ListenAndServe(*apiAddr, srv))
}
