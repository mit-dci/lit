package wallit

// helper functions to test out the new coin selection algo
import (
	"crypto/sha512"
	"encoding/binary"

	"github.com/mit-dci/lit/portxo"
)

type Rand struct {
	buf [sha512.Size]byte
}

// lets create a deterministic prng
// New returns a new seeded Rand.

func (r *Rand) RandNumber() uint32 {
	hash := sha512.New()
	hash.Write(r.buf[:])
	copy(r.buf[:], hash.Sum(nil))
	return binary.LittleEndian.Uint32(r.buf[:])
}

func New(seed uint32) *Rand {
	rnd := Rand{}
	binary.LittleEndian.PutUint32(rnd.buf[:], seed)
	return &rnd
}

func (r *Rand) RandRange(n uint32) (uint32, error) {
	// return a random nuumber in [0,n)
	if n <= 0 {
		return nil, fmt.Errorf("Invalid Argument passed to Rand Range")
	}
	if n&(n-1) == 0 { // n is a power of 2, can mask
		return r.RandNumber() & (n - 1), nil
	}
	// avoid bias: ignore numbers outside of [0, K),
	// where K is a max. 64-bit number such that K % n == 0.
	const maxUint32 = 0xffffffff // we can reduce this based on the size of the array?
	max := maxUint32 - maxUint32%n - 1
	v := r.RandNumber()
	for v > max {
		v = r.RandNumber()
	}
	return v % n, nil
}

func Shuffle(arr portxo.TxoSliceByAmt) (portxo.TxoSliceByAmt, error) {
	if len(arr) == 0 {
		return nil, fmt.Errorf("The length of the utxo pool is zero")
	}

	if len(arr) == 1 {
		return arr, nil
	}

	seed := New(12345)
	for i := range arr { // elcetrum does reverse range, but
		// the for loop turns out to be a bit ugly, so let's do this instead.
		j := seed.RandRange(uint32(len(arr) + 1)) // +1 sicne the last element has to be shuffled
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr, nil
}

func Choice(arr portxo.TxoSliceByAmt) (*portxo.PorTxo, error) {

	if len(arr) == 0 {
		return nil, fmt.Errorf("There is no Slice")
	}

	if len(arr) == 1 { // could remove this, we don't really want this to be here
		return arr[0], nil
	}

	seed := New(12345)
	j := seed.RandRange(uint32(len(arr) + 1))
	return arr[j]
}

// Electrum change policy: max(max(output-txos)*1.25, 0.02) for btc
// question is, do we follow them?
// Stuff to do:
// 1. Penatly function similar to that of Electrum -> Privacy mode only?
// 2. Figure out when they choose the privacy vs random model
// 3. Is the new one better than the old one?
