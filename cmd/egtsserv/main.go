package main

import (
	"flag"
	"github.com/egorban/egtsServ/pkg/egtsserv"
)

func main() {
	var listenPort string
	var numPackets int
	flag.StringVar(&listenPort, "p", "9002", "listen port (e.g. 'localhost:9002')")
	flag.IntVar(&numPackets, "n", 100, "num received packets (e.g. 100)")
	flag.Parse()
	egtsserv.Start(listenPort, numPackets)
}
