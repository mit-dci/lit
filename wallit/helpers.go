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

func SliceSlice(slice []portxo.TxoSliceByAmt, s int) []portxo.TxoSliceByAmt {
	return append(slice[:s], slice[s+1:]...)
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

func getBadness(txos []portxo.PorTxo, sum, amtWanted int64) int32 {

	// penalty stuff, what does the below thing mean?
	// min_change = min(o[2] for o in tx.outputs()) * 0.75
	// max_change = max(o[2] for o in tx.outputs()) * 1.33
	// spent_amount = sum(o[2] for o in tx.outputs())
	min_change := 0 // placeholders
	max_change := 500000000 // placeholders
	spent_amount := 20000 // placeholders
	change := sum - (amtWanted + spentAmount) // change this
	var badness int32
	badness = int32(len(txos) - 1)

	if change < min_change {
		badness += (min_change - change) / (min_change + 10000)
	} else if change > max_change {
		badness += (change - max_change) / (max_change + 10000)
		// Penalize large change; 5 BTC excess ~= using 1 more input
		badness += change / (COIN * 5)
	}
	return badness
}

// while choosing buckets, we must take this badness index into accounting
// in fact, this is our primary driving force behind the whole "privacy" approach
// the winner is chosen adn returned
// we might have to find the place as to where this thing is called

// Electrum change policy: max(max(output-txos)*1.25, 0.02) for btc
// question is, do we follow them?
// Stuff to do:
// 1. Figure out when they choose the privacy vs random model
// 2. Is the new one better than the old one?
