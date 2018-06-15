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

package lyra2rev2

import (
	"github.com/aead/skein"
	"github.com/dchest/blake256"
	"github.com/bitgoin/lyra2rev2/sha3"
)

//Sum returns the result of Lyra2re2 hash.
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
	resultkeccak := keccak.Sum(nil)

	resultcube := cubehash256(resultkeccak)
	lyra2result := make([]byte, 32)
	lyra2(lyra2result, resultcube, resultcube, 1, 4, 4)
	var skeinresult [32]byte
	skein.Sum256(&skeinresult, lyra2result, nil)
	resultcube2 := cubehash256(skeinresult[:])
	resultbmw := bmw256(resultcube2)
	return resultbmw, nil
}
