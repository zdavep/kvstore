package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/zdavep/kvstore/internal/kvstore"
	tmnode "github.com/zdavep/kvstore/internal/tm/node"

	tmdb "github.com/tendermint/tm-db"

	_ "net/http/pprof"
)

var configFile = flag.String("config", "", "Path to config.toml")

func main() {
	flag.Parse()
	if *configFile == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	db := tmdb.NewDB("kvstore", tmdb.CLevelDBBackend, "data/kvstore.db")
	defer db.Close()
	app := kvstore.NewApp(db)
	node, err := tmnode.New(app, *configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(2)
	}
	if err := node.Start(); err != nil {
		panic(err)
	}
	defer func() {
		if err := node.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to stop node: %v", err)
		}
		node.Wait()
	}()
	if os.Getenv("PPROF") != "" {
		go func() {
			if err := http.ListenAndServe(":6060", nil); err != nil {
				fmt.Fprintf(os.Stderr, "failed to start http server: %v", err)
			}
		}()
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(0)
}
