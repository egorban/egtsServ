package main

import (
	"flag"
	"github.com/egorban/egtsServ/pkg/egtsserv"
)

func main() {
	var listenAddress string
	flag.StringVar(&listenAddress, "l", "", "listen address (e.g. 'localhost:8080')")
	flag.Parse()
	egtsserv.Start(listenAddress)
}
