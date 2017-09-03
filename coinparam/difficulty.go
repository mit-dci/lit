package coinparam

import (
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"os"

	"github.com/adiabat/btcd/wire"
)

/* calcDiff returns a bool given two block headers.  This bool is
true if the correct dificulty adjustment is seen in the "next" header.
Only feed it headers n-2016 and n-1, otherwise it will calculate a difficulty
when no adjustment should take place, and return false.
Note that the epoch is actually 2015 blocks long, which is confusing. */
func calcDiffAdjustBitcoin(start, end *wire.BlockHeader, p *Params) uint32 {

	duration := end.Timestamp.Unix() - start.Timestamp.Unix()

	minRetargetTimespan :=
		int64(p.TargetTimespan.Seconds()) / p.RetargetAdjustmentFactor
	maxRetargetTimespan :=
		int64(p.TargetTimespan.Seconds()) * p.RetargetAdjustmentFactor

	if duration < minRetargetTimespan {
		duration = minRetargetTimespan
	} else if duration > maxRetargetTimespan {
		duration = maxRetargetTimespan
	}

	// calculation of new 32-byte difficulty target
	// first turn the previous target into a big int
	prevTarget := CompactToBig(end.Bits)
	// new target is old * duration...
	newTarget := new(big.Int).Mul(prevTarget, big.NewInt(duration))
	// divided by 2 weeks
	newTarget.Div(newTarget, big.NewInt(int64(p.TargetTimespan.Seconds())))

	// clip again if above minimum target (too easy)
	if newTarget.Cmp(p.PowLimit) > 0 {
		newTarget.Set(p.PowLimit)
	}

	// calculate and return 4-byte 'bits' difficulty from 32-byte target
	return BigToCompact(newTarget)
}

// diffBitcoin checks the difficulty of the last header in the slice presented
// give at least an epochlength of headers if this is a new epoch;
// otherwise give at least 2
// it's pretty ugly that it needs Params.  There must be some trick to get
// rid of that since diffBitcoin itself is already in the Params...
func diffBitcoin(
	headers []*wire.BlockHeader, height int32, p *Params) (uint32, error) {

	ltcmode := p.Name == "litetest4" || p.Name == "litereg" ||
		p.Name == "litecoin" || p.Name == "vtctest" || p.Name == "vtc"

	if p.Name == "regtest" {
		return 0x207fffff, nil
	}

	prev := headers[len(headers)-2]
	cur := headers[len(headers)-1]

	// normal, no adjustment; Dn = Dn-1
	rightBits := prev.Bits

	epochLength := int(p.TargetTimespan / p.TargetTimePerBlock)
	epochStart := new(wire.BlockHeader)

	epochHeight := int(height) % epochLength
	maxHeader := len(headers) - 1

	// must include an epoch start header
	if epochHeight > maxHeader {
		return 0, fmt.Errorf("diffBitcoin got insufficient headers")
	}

	epochStart = headers[maxHeader-epochHeight]

	// see if we're on a difficulty adjustment block
	if epochHeight == 0 {
		// if so, we need at least an epoch's worth of headers
		if maxHeader < int(epochLength) {
			return 0, fmt.Errorf("diffBitcoin not enough headers, got %d, need %d",
				len(headers), epochLength)
		}

		if ltcmode {
			if int(height) == epochLength {
				epochStart = headers[maxHeader-epochLength]
			} else {
				epochStart = headers[maxHeader-(epochLength-1)]
			}
		} else {
			epochStart = headers[maxHeader-epochLength]
		}
		// if so, check if difficulty adjustment is valid.
		// That whole "controlled supply" thing.
		// calculate diff n based on n-2016 ... n-1
		rightBits = calcDiffAdjustBitcoin(epochStart, prev, p)
		fmt.Printf("h %d diff adjust %x -> %x\n",
			height, prev.Bits, rightBits)
	} else if p.ReduceMinDifficulty { // not a new epoch
		// if on testnet, check for difficulty nerfing
		if cur.Timestamp.After(
			prev.Timestamp.Add(p.TargetTimePerBlock * 2)) {
			rightBits = p.PowLimitBits // difficulty 1
		} else {
			// actually need to iterate back to last nerfed block,
			// then take the diff from the one behind it
			// btcd code is findPrevTestNetDifficulty()
			// code in bitcoin/cpp:
			// while (pindex->pprev &&
			// pindex->nHeight % params.DifficultyAdjustmentInterval() != 0 &&
			// pindex->nBits == nProofOfWorkLimit)

			// ugh I don't know, and whatever this is testnet.
			// just go to epoch start even though that's not what the cpp code
			// seems to say
			rightBits = epochStart.Bits
		}
	}

	return rightBits, nil
}

func diffBTC(r io.ReadSeeker, height, startheight int32, p *Params) (uint32, error) {
	epochLength := int32(p.TargetTimespan / p.TargetTimePerBlock)
	var err error
	var cur, prev wire.BlockHeader
	ltcmode := p.Name == "litetest4" || p.Name == "litereg" ||
		p.Name == "litecoin" || p.Name == "vtctest" || p.Name == "vtc"
	offsetHeight := height - startheight
	// seek to n-1 header
	_, err = r.Seek(int64(80*(offsetHeight-1)), os.SEEK_SET)
	if err != nil {
		log.Printf(err.Error())
		return 0, err
	}
	// read in n-1
	err = prev.Deserialize(r)
	if err != nil {
		log.Printf(err.Error())
		return 0, err
	}
	// seek to curHeight header and read in
	_, err = r.Seek(int64(80*(offsetHeight)), os.SEEK_SET)
	if err != nil {
		log.Printf(err.Error())
		return 0, err
	}
	err = cur.Deserialize(r)
	if err != nil {
		log.Printf(err.Error())
		return 0, err
	}

	rightBits := prev.Bits // normal, no adjustment; Dn = Dn-1
	// see if we're on a difficulty adjustment block
	if (height)%epochLength == 0 {
		var epochStart wire.BlockHeader
		if ltcmode {
			if height == epochLength {
				_, err = r.Seek(int64(80*(offsetHeight-epochLength)), os.SEEK_SET)
			} else {
				_, err = r.Seek(int64(80*(offsetHeight-epochLength-1)), os.SEEK_SET)
			}
		} else {
			_, err = r.Seek(int64(80*(offsetHeight-epochLength)), os.SEEK_SET)
		}
		if err != nil {
			log.Printf(err.Error())
			return 0, err
		}
		err = epochStart.Deserialize(r)
		if err != nil {
			log.Printf(err.Error())
			return 0, err
		}
		// if so, check if difficulty adjustment is valid.
		// That whole "controlled supply" thing.
		// calculate diff n based on n-2016 ... n-1
		rightBits = calcDiffAdjustBitcoin(&epochStart, &prev, p)
	} else if p.ReduceMinDifficulty { // not a new epoch
		// if on testnet, check for difficulty nerfing
		if cur.Timestamp.After(
			prev.Timestamp.Add(p.TargetTimePerBlock * 2)) {
			rightBits = p.PowLimitBits // difficulty 1
		} else {
			var epochStart wire.BlockHeader
			_, err = r.Seek(int64(80*(offsetHeight-(offsetHeight%epochLength))), os.SEEK_SET)
			if err != nil {
				log.Printf(err.Error())
				return 0, err
			}
			err = epochStart.Deserialize(r)
			if err != nil {
				log.Printf(err.Error())
				return 0, err
			}

			rightBits = epochStart.Bits
		}
	}

	return rightBits, nil
}

func diffKGW(headers []*wire.BlockHeader, height int32, p *Params) (uint32, error) {
	var minBlocks, maxBlocks int32
	minBlocks = 144
	maxBlocks = 4032

	if height-1 < minBlocks {
		return p.PowLimitBits, nil
	}

	currentBlock := headers[len(headers)-2]

	lastSolved := currentBlock

	var blocksScanned, actualRate, targetRate int64
	var difficultyAverage, previousDifficultyAverage big.Int
	var rateAdjustmentRatio, eventHorizonDeviation float64
	var eventHorizonDeviationFast, eventHorizonDevationSlow float64
	rateAdjustmentRatio = 1

	currentHeight := height - 1

	var i int32

	for i = 1; currentHeight > 0; i++ {
		if i > maxBlocks {
			break
		}

		blocksScanned++

		if i == 1 {
			difficultyAverage = *CompactToBig(currentBlock.Bits)
		} else {
			compact := CompactToBig(currentBlock.Bits)

			difference := new(big.Int).Sub(compact, &previousDifficultyAverage)
			difference.Div(difference, big.NewInt(int64(i)))
			difference.Add(difference, &previousDifficultyAverage)
			difficultyAverage = *difference
		}

		previousDifficultyAverage = difficultyAverage

		actualRate = lastSolved.Timestamp.Unix() - currentBlock.Timestamp.Unix()
		targetRate = int64(p.TargetTimePerBlock.Seconds()) * blocksScanned
		rateAdjustmentRatio = 1

		if actualRate < 0 {
			actualRate = 0
		}

		if actualRate != 0 && targetRate != 0 {
			rateAdjustmentRatio = float64(targetRate) / float64(actualRate)
		}

		eventHorizonDeviation = 1 + (0.7084 *
			math.Pow(float64(blocksScanned)/float64(minBlocks), -1.228))
		eventHorizonDeviationFast = eventHorizonDeviation
		eventHorizonDevationSlow = 1 / eventHorizonDeviation

		if blocksScanned >= int64(minBlocks) &&
			(rateAdjustmentRatio <= eventHorizonDevationSlow ||
				rateAdjustmentRatio >= eventHorizonDeviationFast) {
			break
		}

		if currentHeight <= 1 {
			break
		}

		currentHeight--

		currentBlock = headers[len(headers)-int(i+2)]
	}

	newTarget := difficultyAverage
	if actualRate != 0 && targetRate != 0 {
		newTarget.Mul(&newTarget, big.NewInt(actualRate))

		newTarget.Div(&newTarget, big.NewInt(targetRate))
	}

	if newTarget.Cmp(p.PowLimit) == 1 {
		newTarget = *p.PowLimit
	}

	return BigToCompact(&newTarget), nil
}

func diffVTCtest(headers []*wire.BlockHeader, height int32, p *Params) (uint32, error) {
	if height < 2116 {
		return diffBitcoin(headers, height, p)
	}

	// Testnet retargets only every 12 blocks
	if height%12 != 0 {
		prev := headers[len(headers)-2]

		return prev.Bits, nil
	}

	// Run KGW
	return diffKGW(headers, height, p)
}
