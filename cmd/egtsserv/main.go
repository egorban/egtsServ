package main

import (
	"flag"

	"github.com/egorban/egtsServ/pkg/egtsserv"
)

func main() {
	var listenAddress string
	flag.StringVar(&listenAddress, "l", "localhost:9002", "listen address (e.g. 'localhost:9002')")
	flag.Parse()
	egtsserv.Start(listenAddress)
}
