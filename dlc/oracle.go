package dlc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type Oracle struct {
	Idx     uint64   // Index of the oracle for refencing in commands
	A, B, Q [33]byte // public keys of the oracle
	Name    string   // Name of the oracle for display purposes
	Url     string   // Base URL of the oracle, if its REST based (optional)
}

// This manually imports an oracle using the three keys (A, B, Q) concatenated and a name for reference purposes
func (mgr *DlcManager) AddOracle(keys [99]byte, name string) (*Oracle, error) {
	var err error

	o := new(Oracle)
	copy(o.A[:], keys[:33])
	copy(o.B[:], keys[34:66])
	copy(o.Q[:], keys[67:])
	o.Url = ""
	o.Name = name
	err = mgr.SaveOracle(o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

type OracleRestPubkeyResponse struct {
	AHex string `json:"A"`
	BHex string `json:"B"`
	QHex string `json:"Q"`
}

// This imports an oracle using a REST endpoint
func (mgr *DlcManager) ImportOracle(url string, name string) (*Oracle, error) {
	req, err := http.NewRequest("GET", url+"/api/pubkey", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response OracleRestPubkeyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	o := new(Oracle)
	A, err := hex.DecodeString(response.AHex)
	if err != nil {
		return nil, err
	}

	B, err := hex.DecodeString(response.BHex)
	if err != nil {
		return nil, err
	}

	Q, err := hex.DecodeString(response.QHex)
	if err != nil {
		return nil, err
	}

	copy(o.A[:], A[:])
	copy(o.B[:], B[:])
	copy(o.Q[:], Q[:])
	o.Url = url
	o.Name = name
	err = mgr.SaveOracle(o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func OracleFromBuffer(buf *bytes.Buffer) (*Oracle, error) {
	o := new(Oracle)

	copy(o.A[:], buf.Next(33))
	copy(o.B[:], buf.Next(33))
	copy(o.Q[:], buf.Next(33))

	var nameLen uint32
	err := binary.Read(buf, binary.BigEndian, &nameLen)
	if err != nil {
		return nil, err
	}
	o.Name = string(buf.Next(int(nameLen)))

	var urlLen uint32
	err = binary.Read(buf, binary.BigEndian, &urlLen)
	if err != nil {
		return nil, err
	}
	o.Url = string(buf.Next(int(urlLen)))

	return o, nil
}

func OracleFromBytes(b []byte) (*Oracle, error) {
	buf := bytes.NewBuffer(b)
	return OracleFromBuffer(buf)
}

func (self *Oracle) Bytes() []byte {
	var buf bytes.Buffer

	buf.Write(self.A[:])
	buf.Write(self.B[:])
	buf.Write(self.Q[:])

	nameBytes := []byte(self.Name)
	nameLen := uint32(len(nameBytes))
	binary.Write(&buf, binary.BigEndian, nameLen)
	buf.Write(nameBytes)

	urlBytes := []byte(self.Url)
	urlLen := uint32(len(urlBytes))
	binary.Write(&buf, binary.BigEndian, urlLen)
	buf.Write(urlBytes)

	return buf.Bytes()
}
