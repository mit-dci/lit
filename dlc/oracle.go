package dlc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// DlcOracle contains the identifying data of an Oracle
type DlcOracle struct {
	Idx  uint64   // Index of the oracle for refencing in commands
	A    [33]byte // public key of the oracle
	Name string   // Name of the oracle for display purposes
	Url  string   // Base URL of the oracle, if its REST based (optional)
}

// AddOracle manually imports an oracle using the pubkey (A) and a name for reference purposes
func (mgr *DlcManager) AddOracle(key [33]byte, name string) (*DlcOracle, error) {
	var err error

	o := new(DlcOracle)
	o.A = key
	o.Url = ""
	o.Name = name
	err = mgr.SaveOracle(o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

// FindOracleByKey function looks up an oracle based on its key
func (mgr *DlcManager) FindOracleByKey(key [33]byte) (*DlcOracle, error) {
	oracles, err := mgr.ListOracles()
	if err != nil {
		return nil, err
	}

	for _, o := range oracles {
		if bytes.Equal(o.A[:], key[:]) {
			return o, nil
		}
	}

	return nil, fmt.Errorf("Oracle not found")
}

// DlcOracleRestPubkeyResponse is the response format for the REST API that returns the pubkey
type DlcOracleRestPubkeyResponse struct {
	AHex string `json:"A"`
}

// ImportOracle imports an oracle using a REST endpoint. It will save the oracle in the database and give it the passed name.
func (mgr *DlcManager) ImportOracle(url string, name string) (*DlcOracle, error) {
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

	var response DlcOracleRestPubkeyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	o := new(DlcOracle)
	A, err := hex.DecodeString(response.AHex)
	if err != nil {
		return nil, err
	}

	copy(o.A[:], A[:])
	o.Url = url
	o.Name = name
	err = mgr.SaveOracle(o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

// DlcOracleRPointResponse is the response format for the REST API that returns the R-point
type DlcOracleRPointResponse struct {
	RHex string `json:"R"`
}

// FetchRPoint retrieves the R-point based on datafeedID and timestamp (unix epoch) from the REST API of the oracle.
func (o *DlcOracle) FetchRPoint(datafeedId, timestamp uint64) ([33]byte, error) {
	var rPoint [33]byte
	if len(o.Url) == 0 {
		return rPoint, fmt.Errorf("Oracle was not imported from the web - cannot fetch R point. Enter manually using the [dlc contract setrpoint] command")
	}

	req, err := http.NewRequest("GET", o.Url+"/api/rpoint/"+strconv.FormatUint(datafeedId, 10)+"/"+strconv.FormatUint(timestamp, 10), nil)
	if err != nil {
		return rPoint, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return rPoint, err
	}
	defer resp.Body.Close()

	var response DlcOracleRPointResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return rPoint, err
	}

	R, err := hex.DecodeString(response.RHex)
	if err != nil {
		return rPoint, err
	}

	copy(rPoint[:], R[:])
	return rPoint, nil

}

// DlcOracleFromBytes parses a byte array that was serialized using DlcOracle.Bytes() back into a DlcOracle struct
func DlcOracleFromBytes(b []byte) (*DlcOracle, error) {
	buf := bytes.NewBuffer(b)
	o := new(DlcOracle)

	copy(o.A[:], buf.Next(33))

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

// Bytes serializes a DlcOracle struct into a byte array
func (self *DlcOracle) Bytes() []byte {
	var buf bytes.Buffer

	buf.Write(self.A[:])

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
