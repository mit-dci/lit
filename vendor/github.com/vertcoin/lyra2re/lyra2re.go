package lyra2re

import (
	"github.com/aead/skein"
	"github.com/bitgoin/lyra2rev2/sha3"
	"github.com/dchest/blake256"
	"github.com/deedlefake/crypto/groestl256"
)

func Sum(data []byte) ([]byte, error) {
	blake := blake256.New()
	if _, err := blake.Write(data); err != nil {
		return nil, err
	}
	resultBlake := blake.Sum(nil)

	keccak := sha3.NewKeccak256()
	if _, err := keccak.Write(resultBlake); err != nil {
		return nil, err
	}
	resultKeccak := keccak.Sum(nil)

	lyra2Result := make([]byte, 32)
	lyra2(lyra2Result, resultKeccak, resultKeccak, 1, 8, 8)

	var skeinResult [32]byte
	skein.Sum256(&skeinResult, lyra2Result, nil)

	groestlResult := groestl256.Sum(skeinResult[:])

	return groestlResult[:], nil
}
