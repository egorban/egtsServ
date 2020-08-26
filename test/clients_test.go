package test

import (
	"net"
	"reflect"
	"testing"

	"github.com/egorban/navprot/pkg/egts"
)

func TestClient(t *testing.T) {
	tests := []struct {
		name     string
		numPacks uint16
		numRecs  uint16
	}{
		{"onePackOneRec", 1, 1},
		{"onePackSeveralRec", 1, 3},
		{"severalPackOneRec", 3, 1},
		{"severalPackSeveralRec", 3, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			conn, err := net.Dial("tcp", "127.0.0.1:8080")
			if err != nil {
				t.Error("Dial error:      ")
				return
			}

			dataToSend := make([]*egts.Packet, 0, 1)
			wantResponseData := make([]*egts.Packet, 0, 1)

			for i := uint16(0); i < tt.numPacks; i++ {
				dataToSend = append(dataToSend, posData(tt.numRecs, i))
				wantResponseData = append(wantResponseData, responseData(tt.numRecs, i))
			}

			for i, d := range dataToSend {
				packetToSend, _ := d.Form()
				_, err = conn.Write(packetToSend)
				if err != nil {
					t.Errorf("write error = %v", err)
					return
				}
				var response [1024]byte
				_, err = conn.Read(response[:])
				if err != nil {
					t.Errorf("read error = %v", err)
					return
				}
				packetToReceive := new(egts.Packet)
				_, err = packetToReceive.Parse(response[:])
				if err != nil {
					t.Errorf("Parse error = %v", err)
					return
				}
				if !reflect.DeepEqual(packetToReceive, wantResponseData[i]) {
					t.Errorf("Form() gotData = %v, want %v", packetToReceive, wantResponseData[i])
				}
			}
			err = conn.Close()
			if err != nil {
				t.Errorf("connection close error = %v", err)
				return
			}
		})
	}
}

func posData(numRecs uint16, id uint16) *egts.Packet {
	dataPos := egts.PosData{
		Time:    1533570258 - egts.Timestamp20100101utc,
		Lon:     37.782409656276556,
		Lat:     55.62752532903746,
		Bearing: 178,
		Valid:   1,
	}
	dataFuel := egts.FuelData{
		Type: 2,
		Fuel: 2,
	}
	subPos := egts.SubRecord{
		Type: egts.EgtsSrPosData,
		Data: &dataPos,
	}
	subFuel := egts.SubRecord{
		Type: egts.EgtsSrPosData,
		Data: &dataFuel,
	}

	records := make([]*egts.Record, 0, 1)
	for i := uint16(0); i < numRecs; i++ {
		rec := &egts.Record{
			RecNum:  i,
			ID:      uint32(i),
			Service: egts.EgtsTeledataService,
			Data:    []*egts.SubRecord{&subPos, &subFuel},
		}
		records = append(records, rec)
	}

	return &egts.Packet{
		Type:    egts.EgtsPtAppdata,
		ID:      id,
		Records: records, //[]*egts.Record{&rec1,&rec2,&rec3},
		Data:    nil,
	}
}

func responseData(numRecs uint16, id uint16) *egts.Packet {
	data := egts.Response{
		RPID:    id,
		ProcRes: 0,
	}
	subRecords := make([]*egts.SubRecord, 0, 1)
	for i := uint16(0); i < numRecs; i++ {
		subData := egts.Confirmation{
			CRN: i,
			RST: 0,
		}
		subRec := &egts.SubRecord{
			Type: egts.EgtsSrResponse,
			Data: &subData,
		}
		subRecords = append(subRecords, subRec)
	}
	rec := egts.Record{
		RecNum:  id,
		ID:      0,
		Service: egts.EgtsTeledataService,
		Data:    subRecords,
	}
	return &egts.Packet{
		Type:    0,
		ID:      id,
		Records: []*egts.Record{&rec},
		Data:    &data,
	}
}
