package lncore

import (
	"fmt"
	"encoding/json"
	"math/rand"
	"testing"

	//"github.com/mit-dci/lit/bech32"
)

var addrs = []string{
	"ln1jcrlgzng25kxqgz5n5m0xeu5960dl020cnefwk",
	"ln1095qlqa8y3jlav7unuds5twwduucwzvcmj4xnu@1.2.3.4:12345",
	"lnpk1jchkczu7p6g6q3ykcp90rgwd5rfveqc9u03r7325azdcyq89kqdqd0u873@1.2.3.4:12345",
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

func makeRandomAddressData(seed int64) *LnAddressData {

	r := rand.New(rand.NewSource(seed))
	
	ad := &LnAddressData{
		Pkh: nil,
		Pubkey: nil,
		IPAddr: nil,
		Port: LnDefaultPort,
	}

	mkpkhnopk := r.Int() % 4 == 0
	mkipa := r.Int() % 2 == 0
	mkuport := mkipa && r.Int() % 2 == 0

	pkbuf := [32]byte{}
	r.Read(pkbuf[:])

	pkhbech32 := ConvertPubkeyToBech32Pkh(pkbuf[:])
	ad.Pkh = &pkhbech32
	
	if !mkpkhnopk {
		ad.Pubkey = pkbuf[:]
	}

	if mkipa {
		s := fmt.Sprintf("%d.%d.%d.%d", r.Intn(256), r.Intn(256), r.Intn(256), r.Intn(256))
		ad.IPAddr = &s
	}

	if mkipa && mkuport {
		ad.Port = uint16(r.Intn(10000) + 1024)
	}
	
	return ad
}

func TestGenAddresses(t *testing.T) {
	for i := 0; i < 4; i++ {
		raddr := makeRandomAddressData(int64(i + 3)) // +3 so it'll generate all the variations we want
		j, err := json.Marshal(raddr)
		if err != nil {
			fmt.Printf("address jsonify error: %s\n", err.Error())
			t.FailNow()
		}
		fmt.Printf("-----\n%s\n", j)
		fmts, err := DumpAddressFormats(raddr)
		if err != nil {
			fmt.Printf("address dump error: %s\n", err.Error())
			t.FailNow()
		}
		for k, v := range fmts {
			fmt.Printf("%s :\n\t%s\n", k, v)
		}
		fmt.Println("")
	}
}
