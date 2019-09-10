package main

import (
	"flag"
	"github.com/ashirko/navprot/pkg/egts"
	"log"
	"net"
)

func main() {
	var listenAddress string
	flag.StringVar(&listenAddress, "l", "", "listen address (e.g. 'localhost:8080')")
	flag.Parse()
	if listenAddress == "" {
		listenAddress = "127.0.0.1:8080"
	}
	l, err := net.Listen("tcp", listenAddress)
	log.Printf("listening... %s", listenAddress)
	if err != nil {
		log.Printf("error while listening: %s", err)
		return
	}
	connNo := uint64(1)
	for {
		c, err := l.Accept()
		if err != nil {
			log.Printf("error while accepting: %s", err)
		}
		log.Printf("accepted connection %d (%s <-> %s)", connNo, c.RemoteAddr(), c.LocalAddr())
		go handleConnection(c, connNo)
		connNo++
	}
}

func handleConnection(conn net.Conn, connNo uint64) {
	var ansPID uint16
	var ansRID uint16
	var restBuf []byte
	for {
		var buf [1024]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			log.Printf("can't get data from connection %d: %v", connNo, err)
			break
		}
		restBuf = append(restBuf, buf[:n]...)
		for len(restBuf) != 0 {
			egtsPack := new(egts.Packet)
			restBuf, err = egtsPack.Parse(restBuf)
			if err != nil {
				log.Printf(" error while parsing EGTS: %v", err)
				restBuf = []byte{}
				break
			}
			responsePack, err := formResponse(egtsPack, ansPID, ansRID)
			if err != nil {
				log.Printf(" error while form response: %v", err)
				restBuf = []byte{}
				break
			}
			ansPID = (ansPID + 1) & 0xffff
			ansRID = (ansRID + 1) & 0xffff
			_, err = conn.Write(responsePack)
			if err != nil {
				log.Printf(" error while write response to %d: %v", connNo, err)
				restBuf = []byte{}
				break
			}
			writeToDB(egtsPack)
		}
	}
	err := conn.Close()
	if err != nil {
		log.Printf(" error while close connection %d: %v", connNo, err)
	}
}

func formResponse(egtsPack *egts.Packet, ansPID uint16, ansRID uint16) (responsePack []byte, err error) {
	subRecords := make([]*egts.SubRecord, 0, 1)
	for _, rec := range egtsPack.Records {
		subData := egts.Confirmation{
			CRN: rec.RecNum,
			RST: 0,
		}
		sub := &egts.SubRecord{
			Type: egts.EgtsSrResponse,
			Data: &subData,
		}
		subRecords = append(subRecords, sub)
	}
	data := egts.Response{
		RPID:    egtsPack.ID,
		ProcRes: 0,
	}

	rec := egts.Record{
		RecNum:  ansRID,
		Service: egts.EgtsTeledataService,
		Data:    subRecords,
	}
	packetData := &egts.Packet{
		Type:    egts.EgtsPtResponse,
		ID:      ansPID,
		Records: []*egts.Record{&rec},
		Data:    &data,
	}
	responsePack, err = packetData.Form()
	return
}

func writeToDB(egtsPack *egts.Packet) {
	log.Printf("egts pack: %s", egtsPack.String())
}
