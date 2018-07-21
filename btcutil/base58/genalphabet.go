// Copyright (c) 2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

//+build ignore

package main

import (
	"bytes"
	"io"
	log "github.com/mit-dci/lit/logs"
	"os"
	"strconv"
)

var (
	start = []byte(`// Copyright (c) 2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// AUTOGENERATED by genalphabet.go; do not edit.

package base58

const (
	// alphabet is the modified base58 alphabet used by Bitcoin.
	alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	alphabetIdx0 = '1'
)

var b58 = [256]byte{`)

	end = []byte(`}`)

	alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
	tab      = []byte("\t")
	invalid  = []byte("255")
	comma    = []byte(",")
	space    = []byte(" ")
	nl       = []byte("\n")
)

func write(w io.Writer, b []byte) {
	_, err := w.Write(b)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	fi, err := os.Create("alphabet.go")
	if err != nil {
		log.Fatal(err)
	}
	defer fi.Close()

	write(fi, start)
	write(fi, nl)
	for i := byte(0); i < 32; i++ {
		write(fi, tab)
		for j := byte(0); j < 8; j++ {
			idx := bytes.IndexByte(alphabet, i*8+j)
			if idx == -1 {
				write(fi, invalid)
			} else {
				write(fi, strconv.AppendInt(nil, int64(idx), 10))
			}
			write(fi, comma)
			if j != 7 {
				write(fi, space)
			}
		}
		write(fi, nl)
	}
	write(fi, end)
	write(fi, nl)
}
