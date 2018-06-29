// Privacy based solution followed in Electrum
// DESCRIPTION:
// Attempts to better preserve user privacy.  First, if any coin is
// spent from a user address, all coins are.  Compared to spending
// from other addresses to make up an amount, this reduces
// information leakage about sender holdings.  It also helps to
// reduce blockchain UTXO bloat, and reduce future privacy loss that
// would come from reusing that address' remaining UTXOs.  Second, it
// penalizes change that is quite different to the sent amount.
// Third, it penalizes change that is too big
// TODO: is this new model better than the existing one?
// even if it isn't, it atelast follows a "standard" approach

package wallit

import (
	"crypto/sha512"
	"encoding/binary"

	"fmt"
	"github.com/mit-dci/lit/portxo"
	"log"
	"math/rand"
	"sort"
)

type Rand struct {
	buf [sha512.Size]byte
}

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

	seed := New(uint32(rand.Int()))
	for i := range arr { // elcetrum does reverse range
		j, err := seed.RandRange(uint32(len(arr)))
		if err != nil {
			return nil, err
		}
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr, nil
}

func PickOne(arr portxo.TxoSliceByAmt) (*portxo.PorTxo, error) {
	var err error
	arr, err = Shuffle(arr) // Shuffle and pick a random utxo
	if err != nil {
		return nil, fmt.Errorf("There was an erro when shuffling the utxo pile")
	}

	if len(arr) == 0 {
		return nil, fmt.Errorf("There is no Slice")
	}

	if len(arr) == 1 { // could remove this, we don't really want this to be here
		return arr[0], nil
	}
	seed := New(uint32(rand.Int()))
	j, err := seed.RandRange(uint32(len(arr)))
	if err != nil {
		return nil, err
	}
	return arr[j], nil
}

func getBadness(utxos []*portxo.PorTxo, amtWanted, fee int64) (float64, error) {
	var maxChange, minChange float64
	if len(utxos) == 0 {
		return 0, fmt.Errorf("There are no tx's in the utxos list")
	}
	minChangeInt := (utxos[0].Value)
	maxChangeInt := (utxos[0].Value)
	amountSpent := float64(amtWanted + fee)
	totalInput := float64(0)
	for _, utx := range utxos {
		if utx.Value < minChangeInt {
			minChangeInt = utx.Value
		}
		if utx.Value > maxChangeInt {
			maxChangeInt = utx.Value
		}
		totalInput += float64(utx.Value)
	}
	maxChange = float64(maxChangeInt) * 0.9
	minChange = float64(minChangeInt) * 0.5
	// minChange = float64(minChangeInt) * 0.75
	// maxChange = float64(maxChangeInt) * 1.33
	// Electrum's default choices are based on maximizing privacy.
	// however, it seems to bias towards larger utxo values for single utxo tx's
	// because in single input utxos, maxChangeInt = totalInput
	change := totalInput - amountSpent
	badness := float64(len(utxos) - 1)
	if change < minChange {
		// totalinput < minChange + amtWanted + fee, increase badness
		// because I want a certain amount of minimum change. Increase badness
		badness += (minChange - change) / (minChange + 10000)
	} else if change > maxChange {
		// totalInput > maxChange + amtWanted + fee
		// Don't waste a big utxo on this. Increase badness
		badness += (change - maxChange) / (maxChange + 10000)
		// Penalize large change; 5 BTC excess ~= using 1 more input
		badness += change / 5
	}
	return badness, nil
}

func (w *Wallit) PickUtxosRandom(
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
		// skip unconfirmed.  Or de-prioritize? Some option for this...
		//		if utxo.AtHeight == 0 {
		//			continue
		//		}
		//if (txs.Height > 100) &&
		if txs.Mature(curHeight) && (txs.Value > 1) { // why are 0-value outputs a thing..?
			// min value can actually be changed to 20000
			// since the dustCutoff is 20000
			sum += txs.Value
		} else {
			log.Println("tx info:", txs.Height, txs.Value)
			SliceSlice(allUtxos, i) // slice off those guys who aren't confirmed
			i -= 1
		}
	}

	if sum <= amtWanted { // handle this case here, don't wanna go further
		return nil, 0, fmt.Errorf("The sum of utxos is insufficient to pay for the amount")
	}

	allUtxos, err = Shuffle(allUtxos)
	if err != nil {
		log.Printf("There was an error while shuffling the utxo pile")
		return nil, 0, err
	}

	// utxos ready
	// coin selection is super complex, and we can definitely do a lot better
	// here!
	// TODO: anyone who wants to: implement more advanced coin selection algo

	var rSlice portxo.TxoSliceByBip69
	var fee int64
	var badnessList []float64              // badnessList contains a list of all the badness values
	var rSliceRef []portxo.TxoSliceByBip69 // the superset containing the array of utxos
	var remainingRef []int64
	j := 0
	// add utxos until we've had enough
	remaining := amtWanted // remaining is how much is needed on input side
	amountSatisfied := false

	for { // run this only for 100 times as specified
		remaining = amtWanted
		amountSatisfied = false
		rSlice = nil
		i := 0
		for { // hopefully 10000 times will be enough
			utxo, err := PickOne(allUtxos)
			if err != nil {
				return nil, 0, fmt.Errorf("An error occured while picking a tx")
			}
			if ow && utxo.Mode&portxo.FlagTxoWitness == 0 {
				continue // skip non-witness
			}
			// yeah, lets add this utxo!
			rSlice = append(rSlice, utxo)
			sum -= utxo.Value
			remaining -= utxo.Value
			if remaining <= 0 {
				fee = EstFee(rSlice, outputByteSize, feePerByte)
				// subtract fee from returned overshoot.
				// (remaining is negative here)
				remaining += fee

				// done adding utxos if remaining below negative est fee
				if remaining < -fee {
					amountSatisfied = true
				}
			}
			if amountSatisfied {
				// if the amount is satisfied, add it to a superset of utxos
				// do this process about 100 times (electrum), evaluate badness 100 times
				// choose the least among that and return
				break
			}
			if i == 10000 {
				break
			}
			i += 1
		}
		badness, err := getBadness(rSlice, int64(amtWanted), int64(fee)) // check badness for the entire slice
		if err != nil {
			log.Println(err)
			return nil, 0, err
		}
		badnessList = append(badnessList, badness)
		rSliceRef = append(rSliceRef, rSlice)
		remainingRef = append(remainingRef, remaining)
		if j == 99 {
			break // should never come here, doing this to make go happy
		}
		j += 1
	}
	minBadness := badnessList[0]
	var index int
	for i, badness := range badnessList {
		if badness < minBadness {
			minBadness = badness
			index = i
		}
	}
	if !amountSatisfied {
		// default to the old method
		log.Println("Reverting to default coin selection algorithm")
		return w.PickUtxosDefault(amtWanted, outputByteSize, feePerByte, ow)
	}
	sort.Sort(rSliceRef[index]) // send sorted.  This is probably redundant?
	return rSliceRef[index], -remainingRef[index], nil
}
