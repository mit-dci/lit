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

import "encoding/binary"

var iv = []uint32{
	0xEA2BD4B4, 0xCCD6F29F, 0x63117E71,
	0x35481EAE, 0x22512D5B, 0xE5D94E63,
	0x7E624131, 0xF4CC12BE, 0xC2D0B696,
	0x42AF2070, 0xD0720C35, 0x3361DA8C,
	0x28CCECA4, 0x8EF8AD83, 0x4680AC00,
	0x40E5FBAB, 0xD89041C3, 0x6107FBD5,
	0x6C859D41, 0xF0B26679, 0x09392549,
	0x5FA25603, 0x65C892FD, 0x93CB6285,
	0x2AF2B5AE, 0x9E4B4E60, 0x774ABFDD,
	0x85254725, 0x15815AEB, 0x4AB6AAD6,
	0x9CDAF8AF, 0xD6032C0A,
}

//CubeHash is for cubehash.
type CubeHash struct {
	x0 uint32
	x1 uint32
	x2 uint32
	x3 uint32
	x4 uint32
	x5 uint32
	x6 uint32
	x7 uint32
	x8 uint32
	x9 uint32
	xa uint32
	xb uint32
	xc uint32
	xd uint32
	xe uint32
	xf uint32
	xg uint32
	xh uint32
	xi uint32
	xj uint32
	xk uint32
	xl uint32
	xm uint32
	xn uint32
	xo uint32
	xp uint32
	xq uint32
	xr uint32
	xs uint32
	xt uint32
	xu uint32
	xv uint32
}

//NewCubeHash initializes anrd retuns Cubuhash struct.
func NewCubeHash() *CubeHash {
	c := &CubeHash{}
	c.x0 = iv[0]
	c.x1 = iv[1]
	c.x2 = iv[2]
	c.x3 = iv[3]
	c.x4 = iv[4]
	c.x5 = iv[5]
	c.x6 = iv[6]
	c.x7 = iv[7]
	c.x8 = iv[8]
	c.x9 = iv[9]
	c.xa = iv[10]
	c.xb = iv[11]
	c.xc = iv[12]
	c.xd = iv[13]
	c.xe = iv[14]
	c.xf = iv[15]
	c.xg = iv[16]
	c.xh = iv[17]
	c.xi = iv[18]
	c.xj = iv[19]
	c.xk = iv[20]
	c.xl = iv[21]
	c.xm = iv[22]
	c.xn = iv[23]
	c.xo = iv[24]
	c.xp = iv[25]
	c.xq = iv[26]
	c.xr = iv[27]
	c.xs = iv[28]
	c.xt = iv[29]
	c.xu = iv[30]
	c.xv = iv[31]

	return c
}

func (c *CubeHash) inputBlock(data []byte) {
	c.x0 ^= binary.LittleEndian.Uint32(data[0:])
	c.x1 ^= binary.LittleEndian.Uint32(data[4:])
	c.x2 ^= binary.LittleEndian.Uint32(data[8:])
	c.x3 ^= binary.LittleEndian.Uint32(data[12:])
	c.x4 ^= binary.LittleEndian.Uint32(data[16:])
	c.x5 ^= binary.LittleEndian.Uint32(data[20:])
	c.x6 ^= binary.LittleEndian.Uint32(data[24:])
	c.x7 ^= binary.LittleEndian.Uint32(data[28:])
}

func (c *CubeHash) sixteenRounds() {
	for i := 0; i < 8; i++ {
		c.xg = c.x0 + c.xg
		c.x0 = (c.x0 << 7) | (c.x0 >> (32 - 7))
		c.xh = c.x1 + c.xh
		c.x1 = (c.x1 << 7) | (c.x1 >> (32 - 7))
		c.xi = c.x2 + c.xi
		c.x2 = (c.x2 << 7) | (c.x2 >> (32 - 7))
		c.xj = c.x3 + c.xj
		c.x3 = (c.x3 << 7) | (c.x3 >> (32 - 7))
		c.xk = c.x4 + c.xk
		c.x4 = (c.x4 << 7) | (c.x4 >> (32 - 7))
		c.xl = c.x5 + c.xl
		c.x5 = (c.x5 << 7) | (c.x5 >> (32 - 7))
		c.xm = c.x6 + c.xm
		c.x6 = (c.x6 << 7) | (c.x6 >> (32 - 7))
		c.xn = c.x7 + c.xn
		c.x7 = (c.x7 << 7) | (c.x7 >> (32 - 7))
		c.xo = c.x8 + c.xo
		c.x8 = (c.x8 << 7) | (c.x8 >> (32 - 7))
		c.xp = c.x9 + c.xp
		c.x9 = (c.x9 << 7) | (c.x9 >> (32 - 7))
		c.xq = c.xa + c.xq
		c.xa = (c.xa << 7) | (c.xa >> (32 - 7))
		c.xr = c.xb + c.xr
		c.xb = (c.xb << 7) | (c.xb >> (32 - 7))
		c.xs = c.xc + c.xs
		c.xc = (c.xc << 7) | (c.xc >> (32 - 7))
		c.xt = c.xd + c.xt
		c.xd = (c.xd << 7) | (c.xd >> (32 - 7))
		c.xu = c.xe + c.xu
		c.xe = (c.xe << 7) | (c.xe >> (32 - 7))
		c.xv = c.xf + c.xv
		c.xf = (c.xf << 7) | (c.xf >> (32 - 7))
		c.x8 ^= c.xg
		c.x9 ^= c.xh
		c.xa ^= c.xi
		c.xb ^= c.xj
		c.xc ^= c.xk
		c.xd ^= c.xl
		c.xe ^= c.xm
		c.xf ^= c.xn
		c.x0 ^= c.xo
		c.x1 ^= c.xp
		c.x2 ^= c.xq
		c.x3 ^= c.xr
		c.x4 ^= c.xs
		c.x5 ^= c.xt
		c.x6 ^= c.xu
		c.x7 ^= c.xv
		c.xi = c.x8 + c.xi
		c.x8 = (c.x8 << 11) | (c.x8 >> (32 - 11))
		c.xj = c.x9 + c.xj
		c.x9 = (c.x9 << 11) | (c.x9 >> (32 - 11))
		c.xg = c.xa + c.xg
		c.xa = (c.xa << 11) | (c.xa >> (32 - 11))
		c.xh = c.xb + c.xh
		c.xb = (c.xb << 11) | (c.xb >> (32 - 11))
		c.xm = c.xc + c.xm
		c.xc = (c.xc << 11) | (c.xc >> (32 - 11))
		c.xn = c.xd + c.xn
		c.xd = (c.xd << 11) | (c.xd >> (32 - 11))
		c.xk = c.xe + c.xk
		c.xe = (c.xe << 11) | (c.xe >> (32 - 11))
		c.xl = c.xf + c.xl
		c.xf = (c.xf << 11) | (c.xf >> (32 - 11))
		c.xq = c.x0 + c.xq
		c.x0 = (c.x0 << 11) | (c.x0 >> (32 - 11))
		c.xr = c.x1 + c.xr
		c.x1 = (c.x1 << 11) | (c.x1 >> (32 - 11))
		c.xo = c.x2 + c.xo
		c.x2 = (c.x2 << 11) | (c.x2 >> (32 - 11))
		c.xp = c.x3 + c.xp
		c.x3 = (c.x3 << 11) | (c.x3 >> (32 - 11))
		c.xu = c.x4 + c.xu
		c.x4 = (c.x4 << 11) | (c.x4 >> (32 - 11))
		c.xv = c.x5 + c.xv
		c.x5 = (c.x5 << 11) | (c.x5 >> (32 - 11))
		c.xs = c.x6 + c.xs
		c.x6 = (c.x6 << 11) | (c.x6 >> (32 - 11))
		c.xt = c.x7 + c.xt
		c.x7 = (c.x7 << 11) | (c.x7 >> (32 - 11))
		c.xc ^= c.xi
		c.xd ^= c.xj
		c.xe ^= c.xg
		c.xf ^= c.xh
		c.x8 ^= c.xm
		c.x9 ^= c.xn
		c.xa ^= c.xk
		c.xb ^= c.xl
		c.x4 ^= c.xq
		c.x5 ^= c.xr
		c.x6 ^= c.xo
		c.x7 ^= c.xp
		c.x0 ^= c.xu
		c.x1 ^= c.xv
		c.x2 ^= c.xs
		c.x3 ^= c.xt

		c.xj = c.xc + c.xj
		c.xc = (c.xc << 7) | (c.xc >> (32 - 7))
		c.xi = c.xd + c.xi
		c.xd = (c.xd << 7) | (c.xd >> (32 - 7))
		c.xh = c.xe + c.xh
		c.xe = (c.xe << 7) | (c.xe >> (32 - 7))
		c.xg = c.xf + c.xg
		c.xf = (c.xf << 7) | (c.xf >> (32 - 7))
		c.xn = c.x8 + c.xn
		c.x8 = (c.x8 << 7) | (c.x8 >> (32 - 7))
		c.xm = c.x9 + c.xm
		c.x9 = (c.x9 << 7) | (c.x9 >> (32 - 7))
		c.xl = c.xa + c.xl
		c.xa = (c.xa << 7) | (c.xa >> (32 - 7))
		c.xk = c.xb + c.xk
		c.xb = (c.xb << 7) | (c.xb >> (32 - 7))
		c.xr = c.x4 + c.xr
		c.x4 = (c.x4 << 7) | (c.x4 >> (32 - 7))
		c.xq = c.x5 + c.xq
		c.x5 = (c.x5 << 7) | (c.x5 >> (32 - 7))
		c.xp = c.x6 + c.xp
		c.x6 = (c.x6 << 7) | (c.x6 >> (32 - 7))
		c.xo = c.x7 + c.xo
		c.x7 = (c.x7 << 7) | (c.x7 >> (32 - 7))
		c.xv = c.x0 + c.xv
		c.x0 = (c.x0 << 7) | (c.x0 >> (32 - 7))
		c.xu = c.x1 + c.xu
		c.x1 = (c.x1 << 7) | (c.x1 >> (32 - 7))
		c.xt = c.x2 + c.xt
		c.x2 = (c.x2 << 7) | (c.x2 >> (32 - 7))
		c.xs = c.x3 + c.xs
		c.x3 = (c.x3 << 7) | (c.x3 >> (32 - 7))
		c.x4 ^= c.xj
		c.x5 ^= c.xi
		c.x6 ^= c.xh
		c.x7 ^= c.xg
		c.x0 ^= c.xn
		c.x1 ^= c.xm
		c.x2 ^= c.xl
		c.x3 ^= c.xk
		c.xc ^= c.xr
		c.xd ^= c.xq
		c.xe ^= c.xp
		c.xf ^= c.xo
		c.x8 ^= c.xv
		c.x9 ^= c.xu
		c.xa ^= c.xt
		c.xb ^= c.xs
		c.xh = c.x4 + c.xh
		c.x4 = (c.x4 << 11) | (c.x4 >> (32 - 11))
		c.xg = c.x5 + c.xg
		c.x5 = (c.x5 << 11) | (c.x5 >> (32 - 11))
		c.xj = c.x6 + c.xj
		c.x6 = (c.x6 << 11) | (c.x6 >> (32 - 11))
		c.xi = c.x7 + c.xi
		c.x7 = (c.x7 << 11) | (c.x7 >> (32 - 11))
		c.xl = c.x0 + c.xl
		c.x0 = (c.x0 << 11) | (c.x0 >> (32 - 11))
		c.xk = c.x1 + c.xk
		c.x1 = (c.x1 << 11) | (c.x1 >> (32 - 11))
		c.xn = c.x2 + c.xn
		c.x2 = (c.x2 << 11) | (c.x2 >> (32 - 11))
		c.xm = c.x3 + c.xm
		c.x3 = (c.x3 << 11) | (c.x3 >> (32 - 11))
		c.xp = c.xc + c.xp
		c.xc = (c.xc << 11) | (c.xc >> (32 - 11))
		c.xo = c.xd + c.xo
		c.xd = (c.xd << 11) | (c.xd >> (32 - 11))
		c.xr = c.xe + c.xr
		c.xe = (c.xe << 11) | (c.xe >> (32 - 11))
		c.xq = c.xf + c.xq
		c.xf = (c.xf << 11) | (c.xf >> (32 - 11))
		c.xt = c.x8 + c.xt
		c.x8 = (c.x8 << 11) | (c.x8 >> (32 - 11))
		c.xs = c.x9 + c.xs
		c.x9 = (c.x9 << 11) | (c.x9 >> (32 - 11))
		c.xv = c.xa + c.xv
		c.xa = (c.xa << 11) | (c.xa >> (32 - 11))
		c.xu = c.xb + c.xu
		c.xb = (c.xb << 11) | (c.xb >> (32 - 11))
		c.x0 ^= c.xh
		c.x1 ^= c.xg
		c.x2 ^= c.xj
		c.x3 ^= c.xi
		c.x4 ^= c.xl
		c.x5 ^= c.xk
		c.x6 ^= c.xn
		c.x7 ^= c.xm
		c.x8 ^= c.xp
		c.x9 ^= c.xo
		c.xa ^= c.xr
		c.xb ^= c.xq
		c.xc ^= c.xt
		c.xd ^= c.xs
		c.xe ^= c.xv
		c.xf ^= c.xu
	}
}

//cubehash56 calculates cubuhash256.
//length of data must be 32 bytes.
func cubehash256(data []byte) []byte {
	c := NewCubeHash()
	buf := make([]byte, 32)
	buf[0] = 0x80
	c.inputBlock(data)
	c.sixteenRounds()
	c.inputBlock(buf)
	c.sixteenRounds()
	c.xv ^= 1
	for j := 0; j < 10; j++ {
		c.sixteenRounds()
	}
	out := make([]byte, 32)
	binary.LittleEndian.PutUint32(out[0:], c.x0)
	binary.LittleEndian.PutUint32(out[4:], c.x1)
	binary.LittleEndian.PutUint32(out[8:], c.x2)
	binary.LittleEndian.PutUint32(out[12:], c.x3)
	binary.LittleEndian.PutUint32(out[16:], c.x4)
	binary.LittleEndian.PutUint32(out[20:], c.x5)
	binary.LittleEndian.PutUint32(out[24:], c.x6)
	binary.LittleEndian.PutUint32(out[28:], c.x7)
	return out
}
