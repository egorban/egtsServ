package main

import (
	"flag"
	"github.com/egorban/egtsServ/pkg/egtsserv"
)

func main() {
	var listenPort string
	flag.StringVar(&listenPort, "p", "9002", "listen port (e.g. 'localhost:9002')")
	flag.Parse()
	egtsserv.Start(listenPort)
}
