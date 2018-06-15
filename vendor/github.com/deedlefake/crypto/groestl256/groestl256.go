package groestl256

import (
	"encoding/binary"
	"hash"
)

const (
	// Size is the size of a hash in bytes.
	Size = 32

	// BlockSize is the blocksize of a hash.
	BlockSize = 64
)

type context struct {
	buf    [64]byte
	offset int
	state  [8]uint64
	count  uint64
}

// New returns a new hash.Hash that computes Groestl-256 hashes.
func New() hash.Hash {
	ctx := &context{}
	ctx.state[7] = Size * 8
	return ctx
}

func (ctx *context) Write(data []byte) (n int, err error) {
	if len(data) < len(ctx.buf)-ctx.offset {
		copy(ctx.buf[ctx.offset:], data)
		ctx.offset += len(data)
		return len(data), nil
	}

	for len(data) > 0 {
		clen := len(ctx.buf) - ctx.offset
		if clen > len(data) {
			clen = len(data)
		}

		copy(ctx.buf[ctx.offset:], data[:clen])
		ctx.offset += clen
		data = data[clen:]

		if ctx.offset == len(ctx.buf) {
			var g, m [len(ctx.state)]uint64
			for u := range ctx.state {
				m[u] = binary.BigEndian.Uint64(ctx.buf[u<<3:])
				g[u] = m[u] ^ ctx.state[u]
			}

			permSmallP(g[:])
			permSmallQ(m[:])

			for u := range ctx.state {
				ctx.state[u] ^= g[u] ^ m[u]
			}

			ctx.count++
			ctx.offset = 0
		}
	}

	return len(data), nil
}

func (ctx *context) close(dst []byte, ub, n uint64) {
	var pad [72]byte

	z := uint64(0x80) >> n
	pad[0] = uint8((ub & -z) | z)

	padLen := 128 - ctx.offset
	count := ctx.count + 2
	if ctx.offset < 56 {
		padLen = 64 - ctx.offset
		count = ctx.count + 1
	}
	binary.BigEndian.PutUint64(pad[padLen-8:], count)

	ctx.Write(pad[:padLen])

	x := ctx.state
	permSmallP(x[:])

	for u := range x {
		ctx.state[u] ^= x[u]
	}
	for u := 0; u < 4; u++ {
		binary.BigEndian.PutUint64(pad[u<<3:], ctx.state[u+4])
	}

	copy(dst, pad[32-len(dst):])

	*ctx = *New().(*context)

	//ctx.init(uint64(len(dst)) << 3)
}

func (ctx *context) Sum(prev []byte) []byte {
	out := append(prev, make([]byte, Size)...)
	ctx.close(out[len(prev):], 0, 0)
	return out
}

func (ctx *context) Reset() {
	*ctx = *New().(*context)
}

func (ctx *context) Size() int {
	return Size
}

func (ctx *context) BlockSize() int {
	return BlockSize
}

// Sum computes the Groestl-256 hash of data.
func Sum(data []byte) (out [Size]byte) {
	h := New()
	h.Write(data)
	h.Sum(out[:0])
	return
}
