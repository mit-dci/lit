package lnutil

import (
	"bytes"
	"encoding/binary"

	"github.com/mit-dci/lit/logging"
)

// I shouldn't even have to write these...

// int32 to 4 bytes.  Always works.
func I32tB(i int32) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, i)
	return buf.Bytes()
}

// uint32 to 4 bytes.  Always works.
func U32tB(i uint32) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, i)
	return buf.Bytes()
}

// 4 byte slice to uint32.  Returns ffffffff if something doesn't work.
func BtU32(b []byte) uint32 {
	if len(b) != 4 {
		logging.Errorf("Got %x to BtU32 (%d bytes)\n", b, len(b))
		return 0xffffffff
	}
	var i uint32
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &i)
	return i
}

// 4 byte slice to int32.  Returns 7fffffff if something doesn't work.
func BtI32(b []byte) int32 {
	if len(b) != 4 {
		logging.Errorf("Got %x to BtI32 (%d bytes)\n", b, len(b))
		return 0x7fffffff
	}
	var i int32
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &i)
	return i
}

// uint64 to 8 bytes.  Always works.
func U64tB(i uint64) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, i)
	return buf.Bytes()
}

// int64 to 8 bytes.  Always works.
func I64tB(i int64) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, i)
	return buf.Bytes()
}

// 8 bytes to int64 (bitcoin amounts).  returns 7fff... if it doesn't work.
func BtI64(b []byte) int64 {
	if len(b) != 8 {
		logging.Errorf("Got %x to BtI64 (%d bytes)\n", b, len(b))
		return 0x7fffffffffffffff
	}
	var i int64
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &i)
	return i
}

// 8 bytes to uint64.  returns ffff. if it doesn't work.
func BtU64(b []byte) uint64 {
	if len(b) != 8 {
		logging.Errorf("Got %x to BtU64 (%d bytes)\n", b, len(b))
		return 0xffffffffffffffff
	}
	var i uint64
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &i)
	return i
}

// NopeString returns true if the string means "nope"
func NopeString(s string) bool {
	nopes := []string{
		"nope", "no", "n", "false", "0", "nil", "null", "disable", "off", "",
	}
	for _, ts := range nopes {
		if ts == s {
			return true
		}
	}
	return false
}

// YupString returns true if the string means "yup"
func YupString(s string) bool {
	yups := []string{
		"yup", "yes", "y", "true", "1", "ok", "enable", "on",
	}
	for _, ts := range yups {
		if ts == s {
			return true
		}
	}
	return false
}
