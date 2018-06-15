/*
 * Copyright (c) 2016, Shinya Yagyu
 * All rights reserved.
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 * 3. Neither the name of the copyright holder nor the names of its
 *    contributors may be used to endorse or promote products derived from this
 *    software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

//from https://github.com/input-output-hk/scrypto/blob/master/src/main/java/fr/cryptohash/
//under Public Domain CC0 license
//https://github.com/input-output-hk/scrypto/blob/master/COPYING

package lyra2rev2

import (
	"encoding/binary"
)

var initVal = []uint32{
	0x40414243, 0x44454647, 0x48494A4B, 0x4C4D4E4F,
	0x50515253, 0x54555657, 0x58595A5B, 0x5C5D5E5F,
	0x60616263, 0x64656667, 0x68696A6B, 0x6C6D6E6F,
	0x70717273, 0x74757677, 0x78797A7B, 0x7C7D7E7F,
}
var final = []uint32{
	0xaaaaaaa0, 0xaaaaaaa1, 0xaaaaaaa2, 0xaaaaaaa3,
	0xaaaaaaa4, 0xaaaaaaa5, 0xaaaaaaa6, 0xaaaaaaa7,
	0xaaaaaaa8, 0xaaaaaaa9, 0xaaaaaaaa, 0xaaaaaaab,
	0xaaaaaaac, 0xaaaaaaad, 0xaaaaaaae, 0xaaaaaaaf,
}

func circularLeft(x uint32, n uint32) uint32 {
	return (x << n) | (x >> (32 - n))
}

type bmw struct {
	m  []uint32
	h  []uint32
	h2 []uint32
	q  []uint32
}

func new() *bmw {
	b := &bmw{
		m:  make([]uint32, 16),
		h:  make([]uint32, 16),
		h2: make([]uint32, 16),
		q:  make([]uint32, 32),
	}
	copy(b.h, initVal)
	return b
}

func (b *bmw) compress(m []uint32) {
	h := b.h
	q := b.q
	q[0] = ((((m[5] ^ h[5]) - (m[7] ^ h[7]) + (m[10] ^ h[10]) + (m[13] ^ h[13]) + (m[14] ^ h[14])) >> 1) ^ (((m[5] ^ h[5]) - (m[7] ^ h[7]) + (m[10] ^ h[10]) + (m[13] ^ h[13]) + (m[14] ^ h[14])) << 3) ^ circularLeft(((m[5]^h[5])-(m[7]^h[7])+(m[10]^h[10])+(m[13]^h[13])+(m[14]^h[14])), 4) ^ circularLeft(((m[5]^h[5])-(m[7]^h[7])+(m[10]^h[10])+(m[13]^h[13])+(m[14]^h[14])), 19)) + h[1]
	q[1] = ((((m[6] ^ h[6]) - (m[8] ^ h[8]) + (m[11] ^ h[11]) + (m[14] ^ h[14]) - (m[15] ^ h[15])) >> 1) ^ (((m[6] ^ h[6]) - (m[8] ^ h[8]) + (m[11] ^ h[11]) + (m[14] ^ h[14]) - (m[15] ^ h[15])) << 2) ^ circularLeft(((m[6]^h[6])-(m[8]^h[8])+(m[11]^h[11])+(m[14]^h[14])-(m[15]^h[15])), 8) ^ circularLeft(((m[6]^h[6])-(m[8]^h[8])+(m[11]^h[11])+(m[14]^h[14])-(m[15]^h[15])), 23)) + h[2]
	q[2] = ((((m[0] ^ h[0]) + (m[7] ^ h[7]) + (m[9] ^ h[9]) - (m[12] ^ h[12]) + (m[15] ^ h[15])) >> 2) ^ (((m[0] ^ h[0]) + (m[7] ^ h[7]) + (m[9] ^ h[9]) - (m[12] ^ h[12]) + (m[15] ^ h[15])) << 1) ^ circularLeft(((m[0]^h[0])+(m[7]^h[7])+(m[9]^h[9])-(m[12]^h[12])+(m[15]^h[15])), 12) ^ circularLeft(((m[0]^h[0])+(m[7]^h[7])+(m[9]^h[9])-(m[12]^h[12])+(m[15]^h[15])), 25)) + h[3]
	q[3] = ((((m[0] ^ h[0]) - (m[1] ^ h[1]) + (m[8] ^ h[8]) - (m[10] ^ h[10]) + (m[13] ^ h[13])) >> 2) ^ (((m[0] ^ h[0]) - (m[1] ^ h[1]) + (m[8] ^ h[8]) - (m[10] ^ h[10]) + (m[13] ^ h[13])) << 2) ^ circularLeft(((m[0]^h[0])-(m[1]^h[1])+(m[8]^h[8])-(m[10]^h[10])+(m[13]^h[13])), 15) ^ circularLeft(((m[0]^h[0])-(m[1]^h[1])+(m[8]^h[8])-(m[10]^h[10])+(m[13]^h[13])), 29)) + h[4]
	q[4] = ((((m[1] ^ h[1]) + (m[2] ^ h[2]) + (m[9] ^ h[9]) - (m[11] ^ h[11]) - (m[14] ^ h[14])) >> 1) ^ ((m[1] ^ h[1]) + (m[2] ^ h[2]) + (m[9] ^ h[9]) - (m[11] ^ h[11]) - (m[14] ^ h[14]))) + h[5]
	q[5] = ((((m[3] ^ h[3]) - (m[2] ^ h[2]) + (m[10] ^ h[10]) - (m[12] ^ h[12]) + (m[15] ^ h[15])) >> 1) ^ (((m[3] ^ h[3]) - (m[2] ^ h[2]) + (m[10] ^ h[10]) - (m[12] ^ h[12]) + (m[15] ^ h[15])) << 3) ^ circularLeft(((m[3]^h[3])-(m[2]^h[2])+(m[10]^h[10])-(m[12]^h[12])+(m[15]^h[15])), 4) ^ circularLeft(((m[3]^h[3])-(m[2]^h[2])+(m[10]^h[10])-(m[12]^h[12])+(m[15]^h[15])), 19)) + h[6]
	q[6] = ((((m[4] ^ h[4]) - (m[0] ^ h[0]) - (m[3] ^ h[3]) - (m[11] ^ h[11]) + (m[13] ^ h[13])) >> 1) ^ (((m[4] ^ h[4]) - (m[0] ^ h[0]) - (m[3] ^ h[3]) - (m[11] ^ h[11]) + (m[13] ^ h[13])) << 2) ^ circularLeft(((m[4]^h[4])-(m[0]^h[0])-(m[3]^h[3])-(m[11]^h[11])+(m[13]^h[13])), 8) ^ circularLeft(((m[4]^h[4])-(m[0]^h[0])-(m[3]^h[3])-(m[11]^h[11])+(m[13]^h[13])), 23)) + h[7]
	q[7] = ((((m[1] ^ h[1]) - (m[4] ^ h[4]) - (m[5] ^ h[5]) - (m[12] ^ h[12]) - (m[14] ^ h[14])) >> 2) ^ (((m[1] ^ h[1]) - (m[4] ^ h[4]) - (m[5] ^ h[5]) - (m[12] ^ h[12]) - (m[14] ^ h[14])) << 1) ^ circularLeft(((m[1]^h[1])-(m[4]^h[4])-(m[5]^h[5])-(m[12]^h[12])-(m[14]^h[14])), 12) ^ circularLeft(((m[1]^h[1])-(m[4]^h[4])-(m[5]^h[5])-(m[12]^h[12])-(m[14]^h[14])), 25)) + h[8]
	q[8] = ((((m[2] ^ h[2]) - (m[5] ^ h[5]) - (m[6] ^ h[6]) + (m[13] ^ h[13]) - (m[15] ^ h[15])) >> 2) ^ (((m[2] ^ h[2]) - (m[5] ^ h[5]) - (m[6] ^ h[6]) + (m[13] ^ h[13]) - (m[15] ^ h[15])) << 2) ^ circularLeft(((m[2]^h[2])-(m[5]^h[5])-(m[6]^h[6])+(m[13]^h[13])-(m[15]^h[15])), 15) ^ circularLeft(((m[2]^h[2])-(m[5]^h[5])-(m[6]^h[6])+(m[13]^h[13])-(m[15]^h[15])), 29)) + h[9]
	q[9] = ((((m[0] ^ h[0]) - (m[3] ^ h[3]) + (m[6] ^ h[6]) - (m[7] ^ h[7]) + (m[14] ^ h[14])) >> 1) ^ ((m[0] ^ h[0]) - (m[3] ^ h[3]) + (m[6] ^ h[6]) - (m[7] ^ h[7]) + (m[14] ^ h[14]))) + h[10]
	q[10] = ((((m[8] ^ h[8]) - (m[1] ^ h[1]) - (m[4] ^ h[4]) - (m[7] ^ h[7]) + (m[15] ^ h[15])) >> 1) ^ (((m[8] ^ h[8]) - (m[1] ^ h[1]) - (m[4] ^ h[4]) - (m[7] ^ h[7]) + (m[15] ^ h[15])) << 3) ^ circularLeft(((m[8]^h[8])-(m[1]^h[1])-(m[4]^h[4])-(m[7]^h[7])+(m[15]^h[15])), 4) ^ circularLeft(((m[8]^h[8])-(m[1]^h[1])-(m[4]^h[4])-(m[7]^h[7])+(m[15]^h[15])), 19)) + h[11]
	q[11] = ((((m[8] ^ h[8]) - (m[0] ^ h[0]) - (m[2] ^ h[2]) - (m[5] ^ h[5]) + (m[9] ^ h[9])) >> 1) ^ (((m[8] ^ h[8]) - (m[0] ^ h[0]) - (m[2] ^ h[2]) - (m[5] ^ h[5]) + (m[9] ^ h[9])) << 2) ^ circularLeft(((m[8]^h[8])-(m[0]^h[0])-(m[2]^h[2])-(m[5]^h[5])+(m[9]^h[9])), 8) ^ circularLeft(((m[8]^h[8])-(m[0]^h[0])-(m[2]^h[2])-(m[5]^h[5])+(m[9]^h[9])), 23)) + h[12]
	q[12] = ((((m[1] ^ h[1]) + (m[3] ^ h[3]) - (m[6] ^ h[6]) - (m[9] ^ h[9]) + (m[10] ^ h[10])) >> 2) ^ (((m[1] ^ h[1]) + (m[3] ^ h[3]) - (m[6] ^ h[6]) - (m[9] ^ h[9]) + (m[10] ^ h[10])) << 1) ^ circularLeft(((m[1]^h[1])+(m[3]^h[3])-(m[6]^h[6])-(m[9]^h[9])+(m[10]^h[10])), 12) ^ circularLeft(((m[1]^h[1])+(m[3]^h[3])-(m[6]^h[6])-(m[9]^h[9])+(m[10]^h[10])), 25)) + h[13]
	q[13] = ((((m[2] ^ h[2]) + (m[4] ^ h[4]) + (m[7] ^ h[7]) + (m[10] ^ h[10]) + (m[11] ^ h[11])) >> 2) ^ (((m[2] ^ h[2]) + (m[4] ^ h[4]) + (m[7] ^ h[7]) + (m[10] ^ h[10]) + (m[11] ^ h[11])) << 2) ^ circularLeft(((m[2]^h[2])+(m[4]^h[4])+(m[7]^h[7])+(m[10]^h[10])+(m[11]^h[11])), 15) ^ circularLeft(((m[2]^h[2])+(m[4]^h[4])+(m[7]^h[7])+(m[10]^h[10])+(m[11]^h[11])), 29)) + h[14]
	q[14] = ((((m[3] ^ h[3]) - (m[5] ^ h[5]) + (m[8] ^ h[8]) - (m[11] ^ h[11]) - (m[12] ^ h[12])) >> 1) ^ ((m[3] ^ h[3]) - (m[5] ^ h[5]) + (m[8] ^ h[8]) - (m[11] ^ h[11]) - (m[12] ^ h[12]))) + h[15]
	q[15] = ((((m[12] ^ h[12]) - (m[4] ^ h[4]) - (m[6] ^ h[6]) - (m[9] ^ h[9]) + (m[13] ^ h[13])) >> 1) ^ (((m[12] ^ h[12]) - (m[4] ^ h[4]) - (m[6] ^ h[6]) - (m[9] ^ h[9]) + (m[13] ^ h[13])) << 3) ^ circularLeft(((m[12]^h[12])-(m[4]^h[4])-(m[6]^h[6])-(m[9]^h[9])+(m[13]^h[13])), 4) ^ circularLeft(((m[12]^h[12])-(m[4]^h[4])-(m[6]^h[6])-(m[9]^h[9])+(m[13]^h[13])), 19)) + h[0]
	q[16] = (((q[0] >> 1) ^ (q[0] << 2) ^ circularLeft(q[0], 8) ^ circularLeft(q[0], 23)) + ((q[1] >> 2) ^ (q[1] << 1) ^ circularLeft(q[1], 12) ^ circularLeft(q[1], 25)) + ((q[2] >> 2) ^ (q[2] << 2) ^ circularLeft(q[2], 15) ^ circularLeft(q[2], 29)) + ((q[3] >> 1) ^ (q[3] << 3) ^ circularLeft(q[3], 4) ^ circularLeft(q[3], 19)) + ((q[4] >> 1) ^ (q[4] << 2) ^ circularLeft(q[4], 8) ^ circularLeft(q[4], 23)) + ((q[5] >> 2) ^ (q[5] << 1) ^ circularLeft(q[5], 12) ^ circularLeft(q[5], 25)) + ((q[6] >> 2) ^ (q[6] << 2) ^ circularLeft(q[6], 15) ^ circularLeft(q[6], 29)) + ((q[7] >> 1) ^ (q[7] << 3) ^ circularLeft(q[7], 4) ^ circularLeft(q[7], 19)) + ((q[8] >> 1) ^ (q[8] << 2) ^ circularLeft(q[8], 8) ^ circularLeft(q[8], 23)) + ((q[9] >> 2) ^ (q[9] << 1) ^ circularLeft(q[9], 12) ^ circularLeft(q[9], 25)) + ((q[10] >> 2) ^ (q[10] << 2) ^ circularLeft(q[10], 15) ^ circularLeft(q[10], 29)) + ((q[11] >> 1) ^ (q[11] << 3) ^ circularLeft(q[11], 4) ^ circularLeft(q[11], 19)) + ((q[12] >> 1) ^ (q[12] << 2) ^ circularLeft(q[12], 8) ^ circularLeft(q[12], 23)) + ((q[13] >> 2) ^ (q[13] << 1) ^ circularLeft(q[13], 12) ^ circularLeft(q[13], 25)) + ((q[14] >> 2) ^ (q[14] << 2) ^ circularLeft(q[14], 15) ^ circularLeft(q[14], 29)) + ((q[15] >> 1) ^ (q[15] << 3) ^ circularLeft(q[15], 4) ^ circularLeft(q[15], 19)) + ((circularLeft(m[0], 1) + circularLeft(m[3], 4) - circularLeft(m[10], 11) + (16 * 0x05555555)) ^ h[7]))
	q[17] = (((q[1] >> 1) ^ (q[1] << 2) ^ circularLeft(q[1], 8) ^ circularLeft(q[1], 23)) + ((q[2] >> 2) ^ (q[2] << 1) ^ circularLeft(q[2], 12) ^ circularLeft(q[2], 25)) + ((q[3] >> 2) ^ (q[3] << 2) ^ circularLeft(q[3], 15) ^ circularLeft(q[3], 29)) + ((q[4] >> 1) ^ (q[4] << 3) ^ circularLeft(q[4], 4) ^ circularLeft(q[4], 19)) + ((q[5] >> 1) ^ (q[5] << 2) ^ circularLeft(q[5], 8) ^ circularLeft(q[5], 23)) + ((q[6] >> 2) ^ (q[6] << 1) ^ circularLeft(q[6], 12) ^ circularLeft(q[6], 25)) + ((q[7] >> 2) ^ (q[7] << 2) ^ circularLeft(q[7], 15) ^ circularLeft(q[7], 29)) + ((q[8] >> 1) ^ (q[8] << 3) ^ circularLeft(q[8], 4) ^ circularLeft(q[8], 19)) + ((q[9] >> 1) ^ (q[9] << 2) ^ circularLeft(q[9], 8) ^ circularLeft(q[9], 23)) + ((q[10] >> 2) ^ (q[10] << 1) ^ circularLeft(q[10], 12) ^ circularLeft(q[10], 25)) + ((q[11] >> 2) ^ (q[11] << 2) ^ circularLeft(q[11], 15) ^ circularLeft(q[11], 29)) + ((q[12] >> 1) ^ (q[12] << 3) ^ circularLeft(q[12], 4) ^ circularLeft(q[12], 19)) + ((q[13] >> 1) ^ (q[13] << 2) ^ circularLeft(q[13], 8) ^ circularLeft(q[13], 23)) + ((q[14] >> 2) ^ (q[14] << 1) ^ circularLeft(q[14], 12) ^ circularLeft(q[14], 25)) + ((q[15] >> 2) ^ (q[15] << 2) ^ circularLeft(q[15], 15) ^ circularLeft(q[15], 29)) + ((q[16] >> 1) ^ (q[16] << 3) ^ circularLeft(q[16], 4) ^ circularLeft(q[16], 19)) + ((circularLeft(m[1], 2) + circularLeft(m[4], 5) - circularLeft(m[11], 12) + (17 * 0x05555555)) ^ h[8]))
	q[18] = (q[2] + circularLeft(q[3], 3) + q[4] + circularLeft(q[5], 7) + q[6] + circularLeft(q[7], 13) + q[8] + circularLeft(q[9], 16) + q[10] + circularLeft(q[11], 19) + q[12] + circularLeft(q[13], 23) + q[14] + circularLeft(q[15], 27) + ((q[16] >> 1) ^ q[16]) + ((q[17] >> 2) ^ q[17]) + ((circularLeft(m[2], 3) + circularLeft(m[5], 6) - circularLeft(m[12], 13) + (18 * 0x05555555)) ^ h[9]))
	q[19] = (q[3] + circularLeft(q[4], 3) + q[5] + circularLeft(q[6], 7) + q[7] + circularLeft(q[8], 13) + q[9] + circularLeft(q[10], 16) + q[11] + circularLeft(q[12], 19) + q[13] + circularLeft(q[14], 23) + q[15] + circularLeft(q[16], 27) + ((q[17] >> 1) ^ q[17]) + ((q[18] >> 2) ^ q[18]) + ((circularLeft(m[3], 4) + circularLeft(m[6], 7) - circularLeft(m[13], 14) + (19 * 0x05555555)) ^ h[10]))
	q[20] = (q[4] + circularLeft(q[5], 3) + q[6] + circularLeft(q[7], 7) + q[8] + circularLeft(q[9], 13) + q[10] + circularLeft(q[11], 16) + q[12] + circularLeft(q[13], 19) + q[14] + circularLeft(q[15], 23) + q[16] + circularLeft(q[17], 27) + ((q[18] >> 1) ^ q[18]) + ((q[19] >> 2) ^ q[19]) + ((circularLeft(m[4], 5) + circularLeft(m[7], 8) - circularLeft(m[14], 15) + (20 * 0x05555555)) ^ h[11]))
	q[21] = (q[5] + circularLeft(q[6], 3) + q[7] + circularLeft(q[8], 7) + q[9] + circularLeft(q[10], 13) + q[11] + circularLeft(q[12], 16) + q[13] + circularLeft(q[14], 19) + q[15] + circularLeft(q[16], 23) + q[17] + circularLeft(q[18], 27) + ((q[19] >> 1) ^ q[19]) + ((q[20] >> 2) ^ q[20]) + ((circularLeft(m[5], 6) + circularLeft(m[8], 9) - circularLeft(m[15], 16) + (21 * 0x05555555)) ^ h[12]))
	q[22] = (q[6] + circularLeft(q[7], 3) + q[8] + circularLeft(q[9], 7) + q[10] + circularLeft(q[11], 13) + q[12] + circularLeft(q[13], 16) + q[14] + circularLeft(q[15], 19) + q[16] + circularLeft(q[17], 23) + q[18] + circularLeft(q[19], 27) + ((q[20] >> 1) ^ q[20]) + ((q[21] >> 2) ^ q[21]) + ((circularLeft(m[6], 7) + circularLeft(m[9], 10) - circularLeft(m[0], 1) + (22 * 0x05555555)) ^ h[13]))
	q[23] = (q[7] + circularLeft(q[8], 3) + q[9] + circularLeft(q[10], 7) + q[11] + circularLeft(q[12], 13) + q[13] + circularLeft(q[14], 16) + q[15] + circularLeft(q[16], 19) + q[17] + circularLeft(q[18], 23) + q[19] + circularLeft(q[20], 27) + ((q[21] >> 1) ^ q[21]) + ((q[22] >> 2) ^ q[22]) + ((circularLeft(m[7], 8) + circularLeft(m[10], 11) - circularLeft(m[1], 2) + (23 * 0x05555555)) ^ h[14]))
	q[24] = (q[8] + circularLeft(q[9], 3) + q[10] + circularLeft(q[11], 7) + q[12] + circularLeft(q[13], 13) + q[14] + circularLeft(q[15], 16) + q[16] + circularLeft(q[17], 19) + q[18] + circularLeft(q[19], 23) + q[20] + circularLeft(q[21], 27) + ((q[22] >> 1) ^ q[22]) + ((q[23] >> 2) ^ q[23]) + ((circularLeft(m[8], 9) + circularLeft(m[11], 12) - circularLeft(m[2], 3) + (24 * 0x05555555)) ^ h[15]))
	q[25] = (q[9] + circularLeft(q[10], 3) + q[11] + circularLeft(q[12], 7) + q[13] + circularLeft(q[14], 13) + q[15] + circularLeft(q[16], 16) + q[17] + circularLeft(q[18], 19) + q[19] + circularLeft(q[20], 23) + q[21] + circularLeft(q[22], 27) + ((q[23] >> 1) ^ q[23]) + ((q[24] >> 2) ^ q[24]) + ((circularLeft(m[9], 10) + circularLeft(m[12], 13) - circularLeft(m[3], 4) + (25 * 0x05555555)) ^ h[0]))
	q[26] = (q[10] + circularLeft(q[11], 3) + q[12] + circularLeft(q[13], 7) + q[14] + circularLeft(q[15], 13) + q[16] + circularLeft(q[17], 16) + q[18] + circularLeft(q[19], 19) + q[20] + circularLeft(q[21], 23) + q[22] + circularLeft(q[23], 27) + ((q[24] >> 1) ^ q[24]) + ((q[25] >> 2) ^ q[25]) + ((circularLeft(m[10], 11) + circularLeft(m[13], 14) - circularLeft(m[4], 5) + (26 * 0x05555555)) ^ h[1]))
	q[27] = (q[11] + circularLeft(q[12], 3) + q[13] + circularLeft(q[14], 7) + q[15] + circularLeft(q[16], 13) + q[17] + circularLeft(q[18], 16) + q[19] + circularLeft(q[20], 19) + q[21] + circularLeft(q[22], 23) + q[23] + circularLeft(q[24], 27) + ((q[25] >> 1) ^ q[25]) + ((q[26] >> 2) ^ q[26]) + ((circularLeft(m[11], 12) + circularLeft(m[14], 15) - circularLeft(m[5], 6) + (27 * 0x05555555)) ^ h[2]))
	q[28] = (q[12] + circularLeft(q[13], 3) + q[14] + circularLeft(q[15], 7) + q[16] + circularLeft(q[17], 13) + q[18] + circularLeft(q[19], 16) + q[20] + circularLeft(q[21], 19) + q[22] + circularLeft(q[23], 23) + q[24] + circularLeft(q[25], 27) + ((q[26] >> 1) ^ q[26]) + ((q[27] >> 2) ^ q[27]) + ((circularLeft(m[12], 13) + circularLeft(m[15], 16) - circularLeft(m[6], 7) + (28 * 0x05555555)) ^ h[3]))
	q[29] = (q[13] + circularLeft(q[14], 3) + q[15] + circularLeft(q[16], 7) + q[17] + circularLeft(q[18], 13) + q[19] + circularLeft(q[20], 16) + q[21] + circularLeft(q[22], 19) + q[23] + circularLeft(q[24], 23) + q[25] + circularLeft(q[26], 27) + ((q[27] >> 1) ^ q[27]) + ((q[28] >> 2) ^ q[28]) + ((circularLeft(m[13], 14) + circularLeft(m[0], 1) - circularLeft(m[7], 8) + (29 * 0x05555555)) ^ h[4]))
	q[30] = (q[14] + circularLeft(q[15], 3) + q[16] + circularLeft(q[17], 7) + q[18] + circularLeft(q[19], 13) + q[20] + circularLeft(q[21], 16) + q[22] + circularLeft(q[23], 19) + q[24] + circularLeft(q[25], 23) + q[26] + circularLeft(q[27], 27) + ((q[28] >> 1) ^ q[28]) + ((q[29] >> 2) ^ q[29]) + ((circularLeft(m[14], 15) + circularLeft(m[1], 2) - circularLeft(m[8], 9) + (30 * 0x05555555)) ^ h[5]))
	q[31] = (q[15] + circularLeft(q[16], 3) + q[17] + circularLeft(q[18], 7) + q[19] + circularLeft(q[20], 13) + q[21] + circularLeft(q[22], 16) + q[23] + circularLeft(q[24], 19) + q[25] + circularLeft(q[26], 23) + q[27] + circularLeft(q[28], 27) + ((q[29] >> 1) ^ q[29]) + ((q[30] >> 2) ^ q[30]) + ((circularLeft(m[15], 16) + circularLeft(m[2], 3) - circularLeft(m[9], 10) + (31 * 0x05555555)) ^ h[6]))
	xl := q[16] ^ q[17] ^ q[18] ^ q[19] ^ q[20] ^ q[21] ^ q[22] ^ q[23]
	xh := xl ^ q[24] ^ q[25] ^ q[26] ^ q[27] ^ q[28] ^ q[29] ^ q[30] ^ q[31]
	h[0] = ((xh << 5) ^ (q[16] >> 5) ^ m[0]) + (xl ^ q[24] ^ q[0])
	h[1] = ((xh >> 7) ^ (q[17] << 8) ^ m[1]) + (xl ^ q[25] ^ q[1])
	h[2] = ((xh >> 5) ^ (q[18] << 5) ^ m[2]) + (xl ^ q[26] ^ q[2])
	h[3] = ((xh >> 1) ^ (q[19] << 5) ^ m[3]) + (xl ^ q[27] ^ q[3])
	h[4] = ((xh >> 3) ^ (q[20] << 0) ^ m[4]) + (xl ^ q[28] ^ q[4])
	h[5] = ((xh << 6) ^ (q[21] >> 6) ^ m[5]) + (xl ^ q[29] ^ q[5])
	h[6] = ((xh >> 4) ^ (q[22] << 6) ^ m[6]) + (xl ^ q[30] ^ q[6])
	h[7] = ((xh >> 11) ^ (q[23] << 2) ^ m[7]) + (xl ^ q[31] ^ q[7])
	h[8] = circularLeft(h[4], 9) + (xh ^ q[24] ^ m[8]) + ((xl << 8) ^ q[23] ^ q[8])
	h[9] = circularLeft(h[5], 10) + (xh ^ q[25] ^ m[9]) + ((xl >> 6) ^ q[16] ^ q[9])
	h[10] = circularLeft(h[6], 11) + (xh ^ q[26] ^ m[10]) + ((xl << 6) ^ q[17] ^ q[10])
	h[11] = circularLeft(h[7], 12) + (xh ^ q[27] ^ m[11]) + ((xl << 4) ^ q[18] ^ q[11])
	h[12] = circularLeft(h[0], 13) + (xh ^ q[28] ^ m[12]) + ((xl >> 3) ^ q[19] ^ q[12])
	h[13] = circularLeft(h[1], 14) + (xh ^ q[29] ^ m[13]) + ((xl >> 4) ^ q[20] ^ q[13])
	h[14] = circularLeft(h[2], 15) + (xh ^ q[30] ^ m[14]) + ((xl >> 7) ^ q[21] ^ q[14])
	h[15] = circularLeft(h[3], 16) + (xh ^ q[31] ^ m[15]) + ((xl >> 2) ^ q[22] ^ q[15])
}

//bmw256 calculates and returns bmw256 of input.
//length of input must be 32 bytes.
func bmw256(input []byte) []byte {
	b := new()
	buf := make([]byte, 64)
	copy(buf, input)
	buf[len(input)] = 0x80
	bitLen := uint64(len(input)) << 3
	binary.LittleEndian.PutUint64(buf[56:], bitLen)
	for i := 0; i < 16; i++ {
		b.m[i] = binary.LittleEndian.Uint32(buf[i*4:])
	}
	b.compress(b.m)
	b.h, b.h2 = b.h2, b.h
	copy(b.h, final)
	b.compress(b.h2)
	output := make([]byte, 32)
	outlen := len(output) >> 2
	for i := 0; i < outlen; i++ {
		j := 16 - outlen + i
		binary.LittleEndian.PutUint32(output[4*i:], b.h[j])
	}
	return output
}
