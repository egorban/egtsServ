package main

import (
	"flag"

	"github.com/egorban/egtsServ/pkg/egtsserv"
)

func main() {
	var listenAddress string
	flag.StringVar(&listenAddress, "l", "127.0.0.1:9002", "listen address (e.g. '127.0.0.1:9002')")
	flag.Parse()
	egtsserv.Start(listenAddress)
}
