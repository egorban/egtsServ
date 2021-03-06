package egtsserv

import (
	"log"
	"net"
	"time"

	"github.com/ashirko/navprot/pkg/egts"
)

const (
	defaultBufferSize = 1024
	writeTimeout      = 10 * time.Second
	readTimeout       = 180 * time.Second
)

type Stat struct {
	numData  int
	numReply int
}

var chanStat chan Stat

func Start(listenAddress string) {
	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Printf("error while listening: %s", err)
		return
	}
	defer l.Close()
	chanStat = make(chan Stat)
	go waitResult()
	log.Printf("EGTS server was started. Listen address: %v", listenAddress)
	connNo := uint64(1)
	for {
		log.Printf("wait accept...")
		c, err := l.Accept()
		if err != nil {
			log.Printf("error while accepting: %s", err)
			return
		}
		defer c.Close()
		log.Printf("accepted connection %d (%s <-> %s)", connNo, c.RemoteAddr(), c.LocalAddr())
		go handleConnection(c, connNo)
		connNo++
	}
}

func handleConnection(conn net.Conn, connNo uint64) {
	var stat Stat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	var ansPID uint16
	var ansRID uint16
	var restBuf []byte
	for {
		select {
		case <-ticker.C:
			chanStat <- stat
			stat.numData = 0
			stat.numReply = 0
		default:
			err := conn.SetReadDeadline(time.Now().Add(readTimeout))
			if err != nil {
				log.Printf("can't set read deadline %s", err)
			}
			var buf [defaultBufferSize]byte
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
					//log.Printf(" error while parsing EGTS: %v", err)
					break
				}
				stat.numData++
				//log.Printf("receive egts packet: %s", egtsPack.String())
				responsePack, err := formResponse(egtsPack, ansPID, ansRID)
				if err != nil {
					//log.Printf(" error while form response: %v", err)
					break
				}
				_, err = egtsPack.Parse(responsePack)
				if err != nil {
					//log.Printf(" error while parsing response EGTS: %v", err)
					break
				}
				ansPID = (ansPID + 1) & 0xffff
				ansRID = (ansRID + 1) & 0xffff
				err = send(conn, responsePack)
				if err != nil {
					//log.Printf(" error while write response to %d: %v", connNo, err)
					break
				}
				stat.numReply++
				//log.Printf("send reply: %s", egtsPack.String())
			}
		}
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

func send(conn net.Conn, packet []byte) error {
	err := conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err != nil {
		return err
	}
	_, err = conn.Write(packet)
	return err
}

func waitResult() {
	var totalStat Stat
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case stat := <-chanStat:
			totalStat.numData = totalStat.numData + stat.numData
			totalStat.numReply = totalStat.numReply + stat.numReply
		case <-ticker.C:
			log.Printf("last minute: receive data %v, send reply %v", totalStat.numData, totalStat.numReply)
			totalStat.numData = 0
			totalStat.numReply = 0
		}
	}
}
