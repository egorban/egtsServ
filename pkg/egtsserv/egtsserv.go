package egtsserv

import (
	"github.com/ashirko/navprot/pkg/egts"
	"log"
	"net"
	"sync"
	"time"
)

type Result struct {
	numConn    uint64
	numReceive int
}

var (
	running     bool
	muRun       sync.Mutex
	newConnChan chan uint64
	closeConn   chan Result
)

const (
	defaultBufferSize = 1024
	writeTimeout      = 10 * time.Second
	readTimeout       = 180 * time.Second
)

func Start(listenPort string, numPackets int) {
	listenAddress := "localhost:" + listenPort
	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Printf("error while listening: %s", err)
		return
	}
	running = true
	defer l.Close()
	log.Printf("EGTS server was started. Listen address: %v; Number packets to receive: %v",
		listenAddress, numPackets)
	newConnChan = make(chan uint64)
	closeConn = make(chan Result)
	go func() {
		connNo := uint64(1)
		for {
			log.Printf("wait accept...")
			c, err := l.Accept()
			if err != nil {
				muRun.Lock()
				if !running {
					muRun.Unlock()
					return
				}
				muRun.Unlock()
				log.Printf("error while accepting: %s", err)
				return
			}
			defer c.Close()
			log.Printf("accepted connection %d (%s <-> %s)", connNo, c.RemoteAddr(), c.LocalAddr())
			go handleConnection(c, connNo, numPackets)
			newConnChan <- connNo
			connNo++
		}
	}()
	results := waitStop()
	muRun.Lock()
	running = false
	muRun.Unlock()
	log.Printf("EGTS server has completed work")
	for _, r := range results {
		endingReceive := ""
		if r.numReceive > 1 {
			endingReceive = "s"
		}
		log.Printf("For connection %d: received %d data packet"+endingReceive,
			r.numConn, r.numReceive)
	}
}

func waitStop() []Result {
	numsConn := uint64(0)
	results := make([]Result, 0)
	for {
		select {
		case <-newConnChan:
			numsConn++
		case res := <-closeConn:
			results = append(results, res)
			numsConn--
			if numsConn == 0 {
				return results
			}
		}
	}
}

func handleConnection(conn net.Conn, connNo uint64, numPacketsToReceive int) {
	res := Result{connNo, 0}
	var ansPID uint16
	var ansRID uint16
	var restBuf []byte
	for res.numReceive < numPacketsToReceive {
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
				log.Printf(" error while parsing EGTS: %v", err)
				break
			}
			log.Printf("receive egts packet: %s", egtsPack.String())
			res.numReceive++
			responsePack, err := formResponse(egtsPack, ansPID, ansRID)
			if err != nil {
				log.Printf(" error while form response: %v", err)
				break
			}
			_, err = egtsPack.Parse(responsePack)
			if err != nil {
				log.Printf(" error while parsing response EGTS: %v", err)
				break
			}
			ansPID = (ansPID + 1) & 0xffff
			ansRID = (ansRID + 1) & 0xffff
			err = send(conn, responsePack)
			if err != nil {
				log.Printf(" error while write response to %d: %v", connNo, err)
				break
			}
			log.Printf("send reply: %s", egtsPack.String())
		}
	}
	time.Sleep(5 * time.Second)
	err := conn.Close()
	if err != nil {
		log.Printf(" error while close connection %d: %v", connNo, err)
	}
	closeConn <- res
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
