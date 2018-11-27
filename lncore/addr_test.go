package lncore

import (
	"fmt"
	"encoding/json"
	"testing"
)

var addrs = []string{
	"ln1acdef",
	"ln1pmclh89haeswrw0unf8awuyqeu4t2uell58ne@1.2.3.4:12345",
}

func TestAddrVerify(t *testing.T) {
	for _, a := range addrs {
		_, err := ParseLnAddr(a)
		if err != nil {
			fmt.Printf("didn't parse but should have: %s (%s)\n", a, err.Error())
			t.Fail()
		}
	}
}

var notAddrs = []string{
	"asdfasdfasdfasfdadfs",
}

func TestAddrVerifyFail(t *testing.T) {
	for _, a := range notAddrs {
		_, err := ParseLnAddr(a)
		if err == nil {
			fmt.Printf("parsed but should not have: %s\n", a)
			t.Fail()
		}
	}
}

func TestAddrParse(t *testing.T) {
	if false {
		t.FailNow()
	}

	_, _ = json.Marshal(t)
}
