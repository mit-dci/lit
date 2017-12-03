// There is some inheritance from the base "coin" class to derived types in electrum.
// Investigating that.. and they use buckets, not slices argh.
// Structure of how stuff is done right now in Electrum:
// Coin Chooser Base (base class): bucketize_coins, change_amounts, change_outputs, make_tx
// Coin Chooser Random(Base): bucket_candidates, choose_bucket
// Coin Chooser Privacy(Random): penalty_func

package wallit

// helper functions to test out the new coin selection algo
// TODO: is this new model better than the existing one?
// even if it isn't, it atelast follows a "standard" approach
// to coin selection (Electrum)
import (
	"crypto/sha512"
	"encoding/binary"

	"fmt"
	"log"
	"sort"

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
		return 0, fmt.Errorf("Invalid Argument passed to Rand Range")
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

func SliceSlice(slice portxo.TxoSliceByAmt, s int) portxo.TxoSliceByAmt {
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
		j, err := seed.RandRange(uint32(len(arr) + 1)) // +1 sicne the last element has to be shuffled
		if err != nil {
			return nil, err
		}
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
	j, err := seed.RandRange(uint32(len(arr) + 1))
	if err != nil {
		return nil, err
	}
	return arr[j], nil
}

func getBadness(utxos []portxo.PorTxo, txos []portxo.PorTxo, sum, amtWanted int64) (int64, error) {

	if len(utxos) == 0 {
		return 0, fmt.Errorf("There are no tx's in the utxos list")
	}
	min_change := (txos[0].Value)
	max_change := (txos[0].Value)
	spent_amount := int64(0)
	total_input := int64(0)
	for _, tx := range txos {
		if tx.Value < min_change {
			min_change = tx.Value
		}
		if tx.Value > max_change {
			max_change = tx.Value
		}
		spent_amount += tx.Value // sum of total output txos
	}
	for _, utx := range utxos {
		total_input += utx.Value
	}
	min_change2 := float64(min_change) * float64(0.75)
	max_change2 := float64(max_change) * float64(1.33)
	// min_change := 0
	// max_change := 500000000
	// spent_amount := 20000
	change := sum - (amtWanted + int64(spent_amount)) // change this
	var badness int64
	badness = int64(len(utxos) - 1)

	if change < int64(min_change2) {
		badness += (int64(min_change2) - change) / (int64(min_change2) + 10000)
	} else if change > int64(max_change2) {
		badness += (change - int64(max_change2)) / (int64(max_change2) + 10000)
		// Penalize large change; 5 BTC excess ~= using 1 more input
		badness += change / 5
	}
	return badness, nil
}

func (w *Wallit) PickUtxosNew(
	amtWanted, outputByteSize, feePerByte int64,
	ow bool) (portxo.TxoSliceByBip69, int64, error) {

	curHeight, err := w.GetDBSyncHeight()
	if err != nil {
		return nil, 0, err
	}

	var allUtxos portxo.TxoSliceByAmt
	allUtxos, err = w.GetAllUtxos()
	if err != nil {
		return nil, 0, err
	}
	// remove frozen utxos from allUtxo slice.  Iterate backwards / trailing delete
	for i := len(allUtxos) - 1; i >= 0; i-- {
		_, frozen := w.FreezeSet[allUtxos[i].Op]
		if frozen {
			// faster than append, and we're sorting a few lines later anyway
			allUtxos[i] = allUtxos[len(allUtxos)-1] // redundant if at last index
			allUtxos = allUtxos[:len(allUtxos)-1]   // trim last element
		}
	}

	// start with utxos sorted by value and pop off utxos which are greater
	// than the send amount... as long as the next 2 are greater.
	// simple / straightforward coin selection optimization, which tends to make
	// 2 in 2 out

	// smallest and unconfirmed last (because it's reversed)
	sum := int64(0)
	for i, txs := range allUtxos {
		//if (txs.Height > 100) &&
		if txs.Mature(curHeight) {
			sum += txs.Value
		} else {
			log.Println("tx info:", txs.Height, txs.Value)
			SliceSlice(allUtxos, i) // slice off those guys whoaren't confirmed
			i -= 1
		}
	}

	// by here, we have an allutxos set with confirmed inputs, hopefully
	// now we're gonna follow electrum's way of a privacy based solution
	// Privacy based solution followed in Electrum:
	// DESCRIPTION:
	// Attempts to better preserve user privacy.  First, if any coin is
	// spent from a user address, all coins are.  Compared to spending
	// from other addresses to make up an amount, this reduces
	// information leakage about sender holdings.  It also helps to
	// reduce blockchain UTXO bloat, and reduce future privacy loss that
	// would come from reusing that address' remaining UTXOs.  Second, it
	// penalizes change that is quite different to the sent amount.
	// Third, it penalizes change that is too big

	if sum <= amtWanted { // handle this case here, don't wanna go further
		log.Println("SUM:", sum)
		return nil, 0, fmt.Errorf("The sum of utxos is insufficient to pay for the amount")
	}

	sort.Sort(sort.Reverse(allUtxos)) // sort by reverse for convenience, we'll shuffle later anyway.

	// placeholder till here, works cool till now

	// utxos ready
	// coin selection is super complex, and we can definitely do a lot better
	// here!
	// TODO: anyone who wants to: implement more advanced coin selection algo
	// 1. Instead of buckets in ELectrum, we have portxo slices

	// rSlice is the return slice of the utxos which are going into the tx
	var rSlice portxo.TxoSliceByBip69
	// add utxos until we've had enough
	remaining := amtWanted // remaining is how much is needed on input side
	for _, utxo := range allUtxos {
		// skip unconfirmed.  Or de-prioritize? Some option for this...
		//		if utxo.AtHeight == 0 {
		//			continue
		//		}
		if remaining > sum {
			log.Println("This sum of utxos is insufficient to make a tx. Please try again")
			return nil, 0, fmt.Errorf("wanted %d but only %d available.",
				remaining, sum)
		}
		log.Println("The value of the utxo that I get is: ", utxo.Value)
		if !utxo.Mature(curHeight) {
			continue // skip immature or unconfirmed time-locked sh outputs
		}
		if ow && utxo.Mode&portxo.FlagTxoWitness == 0 {
			continue // skip non-witness
		}
		// why are 0-value outputs a thing..?
		if utxo.Value < 1 {
			continue
		}
		// yeah, lets add this utxo!
		rSlice = append(rSlice, utxo)
		sum -= utxo.Value
		remaining -= utxo.Value
		// if remaining is positive, don't bother checking fee yet.
		// if remaining is negative, calculate needed fee
		if remaining <= 0 {
			fee := EstFee(rSlice, outputByteSize, feePerByte)
			// subtract fee from returned overshoot.
			// (remaining is negative here)
			remaining += fee

			// done adding utxos if remaining below negative est fee
			if remaining < -fee {
				break
			}
		}
	}

	if remaining > 0 {
		return nil, 0, fmt.Errorf("wanted %d but %d available.",
			amtWanted, amtWanted-remaining)
		// guy returns negative stuff sometimes
	}

	sort.Sort(rSlice) // send sorted.  This is probably redundant?
	return rSlice, -remaining, nil
}
