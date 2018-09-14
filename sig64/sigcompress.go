package sig64

import "fmt"

// signatures are like 71 or 70 bytes, variable length... this really bugs me!
// But I've lived with it.  But with this, the (txid, sig) pair is slightly over
// 100 bytes, so that bugs me even more, because it'd be really nice to have the
// state data be exactly 100 bytes.  So these functions "compress" the signatures
// into 65 bytes, and restore back into the ~71 byte normal length.

func SigCompress(sig []byte) (csig [64]byte, err error) {
	if len(sig) < 65 || len(sig) > 72 {
		err = fmt.Errorf("Can't compress; sig length is %d", len(sig))
		return
	}
	if sig[0] != 0x30 {
		err = fmt.Errorf("Can't compress; sig starts with %x instead of 30", sig[0])
		return
	}

	// on the web it says sigs can be 72 or even 73 bytes, which I guess means that
	// s can also be 33 bytes?  But I've never seen it.  Maybe 73 is from the
	// sighash byte at the end?  Seems like 65 is enough.

	var r [32]byte
	var s [32]byte

	// pop off first byte of signature which is always 30
	sig = sig[1:]

	// check reported length vs actual length
	if int(sig[0]) != len(sig)-1 {
		err = fmt.Errorf("length byte is %x (%d) but rest is %d bytes long",
			sig[0], sig[0], len(sig)-1)
	}
	// pop off another byte (the length byte)
	sig = sig[1:]

	// 0x02 here just for fun
	if sig[0] != 0x02 {
		err = fmt.Errorf("should have an 0x02 byte, %x instead", sig[0])
		return
	}
	// and pop off that
	sig = sig[1:]

	// get length of R
	rlen := sig[0]
	// pop that off
	sig = sig[1:]
	// check rlen
	if rlen < 22 || rlen > 33 {
		err = fmt.Errorf("%d byte r is too long", rlen)
		return
	}
	if int(rlen) > len(sig) {
		err = fmt.Errorf("length of r value %d but only %d bytes left ",
			rlen, len(sig))
	}
	if rlen == 33 { //drop 0 byte if rlen is 33
		sig = sig[1:]
		rlen = 32
	}
	// copy r, leaving 0s at the MSB
	copy(r[32-rlen:], sig[:rlen])
	// pop off r
	sig = sig[rlen:]

	// 0x02 again, just for fun
	if sig[0] != 0x02 {
		err = fmt.Errorf("should have an 0x02 byte, %x instead", sig[0])
		return
	}
	// and pop off that
	sig = sig[1:]

	// get length of s
	slen := sig[0]
	// pop that off
	sig = sig[1:]
	// check slen
	if slen < 22 || slen > 33 {
		err = fmt.Errorf("%d byte s is too long", slen)
		return
	}
	if int(slen) > len(sig) {
		err = fmt.Errorf("length of s value %d but only %d bytes left ",
			slen, len(sig))
	}
	if slen == 33 { //drop 0 byte if slen is 33
		sig = sig[1:]
		slen = 32
	}
	// copy s, leaving 0s at the MSB
	copy(s[32-slen:], sig[:slen])
	// we're done with sig
	// serialize compressed sig as r, s
	copy(csig[0:32], r[:])
	copy(csig[32:64], s[:])

	return
}

/*
3045 0221 00b7cfe9d300b30f9705633c3b031f8312a189dde2be5aad2e28d73aa617d8ad42 0220 4ff09fd52705fee8733129466e9da417c17b99496360a9a202172b92b8bc78ef
3044 0220   77bd60f213f867e44c85810ebd69e0cf365cf2f20d45264d017b9dde366836d8 0220 00cf16a67d3ca7eee2297e7720e3c077a0a48a7dbbc649bea73d436e2f8c2af9
3045 0221 00bb6300be5526b13f901532c29ce9609e944c1124b13c96f86dd096ac50bafbea 0220 12d39d7a72ea859bd8d31e0a7ea96e2307da100d6ff9c86ab2a83cd17a8ef222
3045 0221 00e9dd01c5cf1b0fbb80af1949def0b226a232acace2a744a01fea73d4897bea9c 0220 526976d608f704e570687dc410edc1c885030ca3aa71f646590e1f0f482fc146
3045 0221 00b4053184157f00bdbf1ff67e411197dd0a46cf410c4abc0d8ea8c3155b0afec7 0220 425ddfe133ee43c3edb99834054db6d46b09b613ec3fea59dc35780fbd9ace72
3044 0220   6048246c95429555d265472d936b71e728f468a84412f9423941b4b9cbbab2f0 0220 4eb1bf82879c72adc3390a638a221792adecf74a097de9bd1257b5bc3e17a407
3045 0221 00b88e7a9137efe437cae4f3dd5e0d05bdf9cf519c10c36d2f657d6bbb0906a50f 0220 41c95b40dc5423864022f5d110a810c50c1dd72c4b679da75e0134f4f5903609
3044 0220   20797760653290231de963a89154a3977685357c2868ae0fb0d241522b62a5c2 0220 211c56e794bf8f04624898094cb4f6beaeff3055bede0c0e5be9cc78b17def22
3044 0220   7652fbfc921747befbcb4193a13b469ddecb767316f1470a01717d758666221f 0220 32131428c3801fd6fa7038b451e7adb4db281e7b38414a8cbc4c604c51574df5

3045 0220   59f28edc62e4b744ff7097717b7d4701614e4af6a30dfa2081ef3e8e27924184 0221 008279ca7eb40a4bd04c923b96110b00d472d648c67df09ad39945130b8f7e4dc8

3044 0220   3af596f8fbeed4c1e6222aecda133c229708c6982c6c98b69a3820331c64d3e0 0220 4f853d4ecea9c7ff0f30fd8e7b47ceddcc0ab2a4d49271062d4e63cc4181354f

3044 0220   57a56687d15830950a8b2b8c89507815d9bbd19d85497f5e68277f03eecb9262 0220 07c74cbb0491adc8bcc1d8dc13286eb5ec54925c582a6bce2b5743e2ccf69750

3045022059f28edc62e4b744ff7097717b7d4701614e4af6a30dfa2081ef3e8e279241840221008279ca7eb40a4bd04c923b96110b00d472d648c67df09ad39945130b8f7e4dc8
304402206048246c95429555d265472d936b71e728f468a84412f9423941b4b9cbbab2f002204eb1bf82879c72adc3390a638a221792adecf74a097de9bd1257b5bc3e17a407
3045022100b88e7a9137efe437cae4f3dd5e0d05bdf9cf519c10c36d2f657d6bbb0906a50f022041c95b40dc5423864022f5d110a810c50c1dd72c4b679da75e0134f4f5903609

*/

func SigDecompress(csig [64]byte) (sig []byte) {
	// this never fails, which is nicer eh?

	// check the beginning of r for 0s.  weird padding.
	rlen := 33
	// chop down to 32 if msb is not set
	if csig[0]&0x80 == 0 {
		rlen--
	}
	// keep chopping down as needed
	i := 0
	for rlen > 1 && csig[i] == 0 && csig[i+1]&0x80 == 0 {
		rlen--
		i++
	}
	padr := make([]byte, rlen)
	copy(padr[rlen-(32-i):], csig[i:])

	// same with s.
	slen := 33
	// chop down to 32 if msb is not set
	if csig[32]&0x80 == 0 {
		slen--
	}
	// keep chopping down as needed
	i = 32
	for slen > 1 && csig[i] == 0 && csig[i+1]&0x80 == 0 {
		slen--
		i++
	}
	pads := make([]byte, slen)
	copy(pads[slen-(64-i):], csig[i:])

	fulllen := byte(len(padr) + len(pads) + 6)
	sig = make([]byte, fulllen)
	sig[0] = 0x30
	sig[1] = fulllen - 2 // the 0x30, and this byte itself
	sig[2] = 0x02
	sig[3] = byte(len(padr))
	// got this copy / offset trick from koblitz/signature.go
	endr := copy(sig[4:], padr) + 4
	sig[endr] = 0x02
	sig[endr+1] = byte(len(pads))
	copy(sig[endr+2:], pads)

	return
}
