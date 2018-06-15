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

/*
Codes comes from https://github.com/monacoinproject/monacoin/tree/8ef4720a7f1f47f937da115c2a0c7ec93b21f7f2/src/Lyra2RE
under MIT license.
https://github.com/monacoinproject/monacoin/blob/master-0.10/COPYING
*/

package lyra2rev2

import "encoding/binary"

const (
	blockLenInt64 = 12                //Block length: 768 bits (=96 bytes, =12 uint64_t)
	blockLenBytes = blockLenInt64 * 8 //Block length, in bytes

	blockLenBlake2SafeInt64 = 8                             //512 bits (=64 bytes, =8 uint64_t)
	blockLenBlake2SafeBytes = (blockLenBlake2SafeInt64 * 8) //same as above, in bytes
)

var blake2bIV = []uint64{
	0x6a09e667f3bcc908, 0xbb67ae8584caa73b,
	0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1,
	0x510e527fade682d1, 0x9b05688c2b3e6c1f,
	0x1f83d9abfb41bd6b, 0x5be0cd19137e2179,
}

/*Blake2b's rotation*/
func rotr64(w uint64, c byte) uint64 {
	return (w >> c) | (w << (64 - c))
}

/*g is Blake2b's G function*/
func g(a, b, c, d uint64) (uint64, uint64, uint64, uint64) {
	a = a + b
	d = rotr64(d^a, 32)
	c = c + d
	b = rotr64(b^c, 24)
	a = a + b
	d = rotr64(d^a, 16)
	c = c + d
	b = rotr64(b^c, 63)
	return a, b, c, d
}

/*roundLyra is One Round of the Blake2b's compression function*/
func roundLyra(v []uint64) {
	v[0], v[4], v[8], v[12] = g(v[0], v[4], v[8], v[12])
	v[1], v[5], v[9], v[13] = g(v[1], v[5], v[9], v[13])
	v[2], v[6], v[10], v[14] = g(v[2], v[6], v[10], v[14])
	v[3], v[7], v[11], v[15] = g(v[3], v[7], v[11], v[15])
	v[0], v[5], v[10], v[15] = g(v[0], v[5], v[10], v[15])
	v[1], v[6], v[11], v[12] = g(v[1], v[6], v[11], v[12])
	v[2], v[7], v[8], v[13] = g(v[2], v[7], v[8], v[13])
	v[3], v[4], v[9], v[14] = g(v[3], v[4], v[9], v[14])
}

/**
 * initState Initializes the Sponge State. The first 512 bits are set to zeros and the remainder
 * receive Blake2b's IV as per Blake2b's specification. <b>Note:</b> Even though sponges
 * typically have their internal state initialized with zeros, Blake2b's G function
 * has a fixed point: if the internal state and message are both filled with zeros. the
 * resulting permutation will always be a block filled with zeros; this happens because
 * Blake2b does not use the constants originally employed in Blake2 inside its G function,
 * relying on the IV for avoiding possible fixed points.
 *
 * @param state         The 1024-bit array to be initialized
 */
func initState() []uint64 {
	state := make([]uint64, 16)
	state[8] = blake2bIV[0]
	state[9] = blake2bIV[1]
	state[10] = blake2bIV[2]
	state[11] = blake2bIV[3]
	state[12] = blake2bIV[4]
	state[13] = blake2bIV[5]
	state[14] = blake2bIV[6]
	state[15] = blake2bIV[7]
	return state
}

/**
 * Eblake2bLyraxecute Blake2b's G function, with all 12 rounds.
 *
 * @param v     A 1024-bit (16 uint64_t) array to be processed by Blake2b's G function
 */
func blake2bLyra(v []uint64) {
	for i := 0; i < 12; i++ {
		roundLyra(v)
	}
}

/**
 * reducedBlake2bLyra Executes a reduced version of Blake2b's G function with only one round
 * @param v     A 1024-bit (16 uint64_t) array to be processed by Blake2b's G function
 */
func reducedBlake2bLyra(v []uint64) {
	roundLyra(v)
}

/**
 * squeeze Performs a squeeze operation, using Blake2b's G function as the
 * internal permutation
 *
 * @param state      The current state of the sponge
 * @param out        Array that will receive the data squeezed
 * @param len        The number of bytes to be squeezed into the "out" array
 */
func squeeze(state []uint64, out []byte) {
	tmp := make([]byte, blockLenBytes)
	for j := 0; j < len(out)/blockLenBytes+1; j++ {
		for i := 0; i < blockLenInt64; i++ {
			binary.LittleEndian.PutUint64(tmp[i*8:], state[i])
		}
		copy(out[j*blockLenBytes:], tmp) //be care in case of len(out[i:])<len(tmp)
		blake2bLyra(state)
	}
}

/**
 * absorbBlock Performs an absorb operation for a single block (BLOCK_LEN_INT64 words
 * of type uint64_t), using Blake2b's G function as the internal permutation
 *
 * @param state The current state of the sponge
 * @param in    The block to be absorbed (BLOCK_LEN_INT64 words)
 */
func absorbBlock(state []uint64, in []uint64) {
	//XORs the first BLOCK_LEN_INT64 words of "in" with the current state
	state[0] ^= in[0]
	state[1] ^= in[1]
	state[2] ^= in[2]
	state[3] ^= in[3]
	state[4] ^= in[4]
	state[5] ^= in[5]
	state[6] ^= in[6]
	state[7] ^= in[7]
	state[8] ^= in[8]
	state[9] ^= in[9]
	state[10] ^= in[10]
	state[11] ^= in[11]

	//Applies the transformation f to the sponge's state
	blake2bLyra(state)
}

/**
 * absorbBlockBlake2Safe  Performs an absorb operation for a single block (BLOCK_LEN_BLAKE2_SAFE_INT64
 * words of type uint64_t), using Blake2b's G function as the internal permutation
 *
 * @param state The current state of the sponge
 * @param in    The block to be absorbed (BLOCK_LEN_BLAKE2_SAFE_INT64 words)
 */
func absorbBlockBlake2Safe(state []uint64, in []uint64) {
	//XORs the first BLOCK_LEN_BLAKE2_SAFE_INT64 words of "in" with the current state

	state[0] ^= in[0]
	state[1] ^= in[1]
	state[2] ^= in[2]
	state[3] ^= in[3]
	state[4] ^= in[4]
	state[5] ^= in[5]
	state[6] ^= in[6]
	state[7] ^= in[7]

	//Applies the transformation f to the sponge's state
	blake2bLyra(state)

}

/**
 * reducedSqueezeRow0 erforms a reduced squeeze operation for a single row, from the highest to
 * the lowest index, using the reduced-round Blake2b's G function as the
 * internal permutation
 *
 * @param state     The current state of the sponge
 * @param rowOut    Row to receive the data squeezed
 */
func reducedSqueezeRow0(state []uint64, rowOut []uint64, nCols int) {
	ptr := (nCols - 1) * blockLenInt64
	//M[row][C-1-col] = H.reduced_squeeze()
	for i := 0; i < nCols; i++ {
		ptrWord := rowOut[ptr:] //In Lyra2: pointer to M[0][C-1]
		ptrWord[0] = state[0]
		ptrWord[1] = state[1]
		ptrWord[2] = state[2]
		ptrWord[3] = state[3]
		ptrWord[4] = state[4]
		ptrWord[5] = state[5]
		ptrWord[6] = state[6]
		ptrWord[7] = state[7]
		ptrWord[8] = state[8]
		ptrWord[9] = state[9]
		ptrWord[10] = state[10]
		ptrWord[11] = state[11]

		//Goes to next block (column) that will receive the squeezed data
		ptr -= blockLenInt64

		//Applies the reduced-round transformation f to the sponge's state
		reducedBlake2bLyra(state)
	}
}

/**
 * reducedDuplexRow1 Performs a reduced duplex operation for a single row, from the highest to
 * the lowest index, using the reduced-round Blake2b's G function as the
 * internal permutation
 *
 * @param state		The current state of the sponge
 * @param rowIn		Row to feed the sponge
 * @param rowOut	Row to receive the sponge's output
 */
func reducedDuplexRow1(state []uint64, rowIn []uint64, rowOut []uint64, nCols int) {
	ptrIn := 0
	ptrOut := (nCols - 1) * blockLenInt64

	for i := 0; i < nCols; i++ {
		ptrWordIn := rowIn[ptrIn:]    //In Lyra2: pointer to prev
		ptrWordOut := rowOut[ptrOut:] //In Lyra2: pointer to row
		//Absorbing "M[prev][col]"
		state[0] ^= (ptrWordIn[0])
		state[1] ^= (ptrWordIn[1])
		state[2] ^= (ptrWordIn[2])
		state[3] ^= (ptrWordIn[3])
		state[4] ^= (ptrWordIn[4])
		state[5] ^= (ptrWordIn[5])
		state[6] ^= (ptrWordIn[6])
		state[7] ^= (ptrWordIn[7])
		state[8] ^= (ptrWordIn[8])
		state[9] ^= (ptrWordIn[9])
		state[10] ^= (ptrWordIn[10])
		state[11] ^= (ptrWordIn[11])

		//Applies the reduced-round transformation f to the sponge's state
		reducedBlake2bLyra(state)

		//M[row][C-1-col] = M[prev][col] XOR rand
		ptrWordOut[0] = ptrWordIn[0] ^ state[0]
		ptrWordOut[1] = ptrWordIn[1] ^ state[1]
		ptrWordOut[2] = ptrWordIn[2] ^ state[2]
		ptrWordOut[3] = ptrWordIn[3] ^ state[3]
		ptrWordOut[4] = ptrWordIn[4] ^ state[4]
		ptrWordOut[5] = ptrWordIn[5] ^ state[5]
		ptrWordOut[6] = ptrWordIn[6] ^ state[6]
		ptrWordOut[7] = ptrWordIn[7] ^ state[7]
		ptrWordOut[8] = ptrWordIn[8] ^ state[8]
		ptrWordOut[9] = ptrWordIn[9] ^ state[9]
		ptrWordOut[10] = ptrWordIn[10] ^ state[10]
		ptrWordOut[11] = ptrWordIn[11] ^ state[11]

		//Input: next column (i.e., next block in sequence)
		ptrIn += blockLenInt64
		//Output: goes to previous column
		ptrOut -= blockLenInt64
	}
}

/**
 * reducedDuplexRowSetup Performs a duplexing operation over "M[rowInOut][col] [+] M[rowIn][col]" (i.e.,
 * the wordwise addition of two columns, ignoring carries between words). The
 * output of this operation, "rand", is then used to make
 * "M[rowOut][(N_COLS-1)-col] = M[rowIn][col] XOR rand" and
 * "M[rowInOut][col] =  M[rowInOut][col] XOR rotW(rand)", where rotW is a 64-bit
 * rotation to the left and N_COLS is a system parameter.
 *
 * @param state          The current state of the sponge
 * @param rowIn          Row used only as input
 * @param rowInOut       Row used as input and to receive output after rotation
 * @param rowOut         Row receiving the output
 *
 */
func reducedDuplexRowSetup(state []uint64, rowIn []uint64,
	rowInOut []uint64, rowOut []uint64, nCols int) {
	ptrIn := 0
	ptrInOut := 0
	ptrOut := (nCols - 1) * blockLenInt64

	for i := 0; i < nCols; i++ {
		ptrWordIn := rowIn[ptrIn:]          //In Lyra2: pointer to prev
		ptrWordOut := rowOut[ptrOut:]       //In Lyra2: pointer to row
		ptrWordInOut := rowInOut[ptrInOut:] //In Lyra2: pointer to row

		//Absorbing "M[prev] [+] M[row*]"
		state[0] ^= (ptrWordIn[0] + ptrWordInOut[0])
		state[1] ^= (ptrWordIn[1] + ptrWordInOut[1])
		state[2] ^= (ptrWordIn[2] + ptrWordInOut[2])
		state[3] ^= (ptrWordIn[3] + ptrWordInOut[3])
		state[4] ^= (ptrWordIn[4] + ptrWordInOut[4])
		state[5] ^= (ptrWordIn[5] + ptrWordInOut[5])
		state[6] ^= (ptrWordIn[6] + ptrWordInOut[6])
		state[7] ^= (ptrWordIn[7] + ptrWordInOut[7])
		state[8] ^= (ptrWordIn[8] + ptrWordInOut[8])
		state[9] ^= (ptrWordIn[9] + ptrWordInOut[9])
		state[10] ^= (ptrWordIn[10] + ptrWordInOut[10])
		state[11] ^= (ptrWordIn[11] + ptrWordInOut[11])

		//Applies the reduced-round transformation f to the sponge's state
		reducedBlake2bLyra(state)

		//M[row][col] = M[prev][col] XOR rand
		ptrWordOut[0] = ptrWordIn[0] ^ state[0]
		ptrWordOut[1] = ptrWordIn[1] ^ state[1]
		ptrWordOut[2] = ptrWordIn[2] ^ state[2]
		ptrWordOut[3] = ptrWordIn[3] ^ state[3]
		ptrWordOut[4] = ptrWordIn[4] ^ state[4]
		ptrWordOut[5] = ptrWordIn[5] ^ state[5]
		ptrWordOut[6] = ptrWordIn[6] ^ state[6]
		ptrWordOut[7] = ptrWordIn[7] ^ state[7]
		ptrWordOut[8] = ptrWordIn[8] ^ state[8]
		ptrWordOut[9] = ptrWordIn[9] ^ state[9]
		ptrWordOut[10] = ptrWordIn[10] ^ state[10]
		ptrWordOut[11] = ptrWordIn[11] ^ state[11]

		//M[row*][col] = M[row*][col] XOR rotW(rand)
		ptrWordInOut[0] ^= state[11]
		ptrWordInOut[1] ^= state[0]
		ptrWordInOut[2] ^= state[1]
		ptrWordInOut[3] ^= state[2]
		ptrWordInOut[4] ^= state[3]
		ptrWordInOut[5] ^= state[4]
		ptrWordInOut[6] ^= state[5]
		ptrWordInOut[7] ^= state[6]
		ptrWordInOut[8] ^= state[7]
		ptrWordInOut[9] ^= state[8]
		ptrWordInOut[10] ^= state[9]
		ptrWordInOut[11] ^= state[10]

		//Inputs: next column (i.e., next block in sequence)
		ptrInOut += blockLenInt64
		ptrIn += blockLenInt64
		//Output: goes to previous column
		ptrOut -= blockLenInt64
	}
}

/**
 * reducedDuplexRow Performs a duplexing operation over "M[rowInOut][col] [+] M[rowIn][col]" (i.e.,
 * the wordwise addition of two columns, ignoring carries between words). The
 * output of this operation, "rand", is then used to make
 * "M[rowOut][col] = M[rowOut][col] XOR rand" and
 * "M[rowInOut][col] =  M[rowInOut][col] XOR rotW(rand)", where rotW is a 64-bit
 * rotation to the left.
 *
 * @param state          The current state of the sponge
 * @param rowIn          Row used only as input
 * @param rowInOut       Row used as input and to receive output after rotation
 * @param rowOut         Row receiving the output
 *
 */
func reducedDuplexRow(state []uint64, rowIn []uint64, rowInOut []uint64,
	rowOut []uint64, nCols int) {
	ptrIn := 0
	ptrInOut := 0
	ptrOut := 0
	for i := 0; i < nCols; i++ {
		ptrWordIn := rowIn[ptrIn:]          //In Lyra2: pointer to prev
		ptrWordOut := rowOut[ptrOut:]       //In Lyra2: pointer to row
		ptrWordInOut := rowInOut[ptrInOut:] //In Lyra2: pointer to row
		//Absorbing "M[prev] [+] M[row*]"
		state[0] ^= (ptrWordIn[0] + ptrWordInOut[0])
		state[1] ^= (ptrWordIn[1] + ptrWordInOut[1])
		state[2] ^= (ptrWordIn[2] + ptrWordInOut[2])
		state[3] ^= (ptrWordIn[3] + ptrWordInOut[3])
		state[4] ^= (ptrWordIn[4] + ptrWordInOut[4])
		state[5] ^= (ptrWordIn[5] + ptrWordInOut[5])
		state[6] ^= (ptrWordIn[6] + ptrWordInOut[6])
		state[7] ^= (ptrWordIn[7] + ptrWordInOut[7])
		state[8] ^= (ptrWordIn[8] + ptrWordInOut[8])
		state[9] ^= (ptrWordIn[9] + ptrWordInOut[9])
		state[10] ^= (ptrWordIn[10] + ptrWordInOut[10])
		state[11] ^= (ptrWordIn[11] + ptrWordInOut[11])

		//Applies the reduced-round transformation f to the sponge's state
		reducedBlake2bLyra(state)

		//M[rowOut][col] = M[rowOut][col] XOR rand
		ptrWordOut[0] ^= state[0]
		ptrWordOut[1] ^= state[1]
		ptrWordOut[2] ^= state[2]
		ptrWordOut[3] ^= state[3]
		ptrWordOut[4] ^= state[4]
		ptrWordOut[5] ^= state[5]
		ptrWordOut[6] ^= state[6]
		ptrWordOut[7] ^= state[7]
		ptrWordOut[8] ^= state[8]
		ptrWordOut[9] ^= state[9]
		ptrWordOut[10] ^= state[10]
		ptrWordOut[11] ^= state[11]

		//M[rowInOut][col] = M[rowInOut][col] XOR rotW(rand)
		ptrWordInOut[0] ^= state[11]
		ptrWordInOut[1] ^= state[0]
		ptrWordInOut[2] ^= state[1]
		ptrWordInOut[3] ^= state[2]
		ptrWordInOut[4] ^= state[3]
		ptrWordInOut[5] ^= state[4]
		ptrWordInOut[6] ^= state[5]
		ptrWordInOut[7] ^= state[6]
		ptrWordInOut[8] ^= state[7]
		ptrWordInOut[9] ^= state[8]
		ptrWordInOut[10] ^= state[9]
		ptrWordInOut[11] ^= state[10]

		//Goes to next block
		ptrOut += blockLenInt64
		ptrInOut += blockLenInt64
		ptrIn += blockLenInt64
	}
}

// lyra2 Executes Lyra2 based on the G function from Blake2b. This version supports salts and passwords
// whose combined length is smaller than the size of the memory matrix, (i.e., (nRows x nCols x b) bits,
// where "b" is the underlying sponge's bitrate). In this implementation, the "basil" is composed by all
// integer parameters (treated as type "unsigned int") in the order they are provided, plus the value
// of nCols, (i.e., basil = kLen || pwdlen || saltlen || timeCost || nRows || nCols).
//
// @param K The derived key to be output by the algorithm
// @param kLen Desired key length
// @param pwd User password
// @param pwdlen Password length
// @param salt Salt
// @param saltlen Salt length
// @param timeCost Parameter to determine the processing time (T)
// @param nRows Number or rows of the memory matrix (R)
// @param nCols Number of columns of the memory matrix (C)
//
// @return 0 if the key is generated correctly; -1 if there is an error (usually due to lack of memory for allocation)
func lyra2(k []byte, pwd []byte, salt []byte, timeCost uint64, nRows int, nCols int) {

	//============================= Basic variables ============================//
	row := 2              //index of row to be processed
	prev := 1             //index of prev (last row ever computed/modified)
	var rowa uint64       //index of row* (a previous row, deterministically picked during Setup and randomly picked while Wandering)
	var tau uint64        //Time Loop iterator
	step := 1             //Visitation step (used during Setup and Wandering phases)
	var window uint64 = 2 //Visitation window (used to define which rows can be revisited during Setup)
	var gap uint64 = 1    //Modifier to the step, assuming the values 1 or -1
	var i int             //auxiliary iteration counter
	//==========================================================================/

	//========== Initializing the Memory Matrix and pointers to it =============//
	//Tries to allocate enough space for the whole memory matrix

	rowLenInt64 := blockLenInt64 * nCols
	//rowLenBytes := rowLenInt64 * 8

	i = nRows * rowLenInt64
	wholeMatrix := make([]uint64, i)
	//Allocates pointers to each row of the matrix
	memMatrix := make([][]uint64, nRows)

	//Places the pointers in the correct positions
	ptrWord := 0
	for i = 0; i < nRows; i++ {
		memMatrix[i] = wholeMatrix[ptrWord:]
		ptrWord += rowLenInt64
	}
	//==========================================================================/

	//============= Getting the password + salt + basil padded with 10*1 ===============//
	//OBS.:The memory matrix will temporarily hold the password: not for saving memory,
	//but this ensures that the password copied locally will be overwritten as soon as possible

	//First, we clean enough blocks for the password, salt, basil and padding
	nBlocksInput := ((len(salt) + len(pwd) + 6*8) / blockLenBlake2SafeBytes) + 1
	ptrByte := 0 // (byte*) wholeMatrix;

	//Prepends the password
	for j := 0; j < len(pwd)/8; j++ {
		wholeMatrix[ptrByte+j] = binary.LittleEndian.Uint64(pwd[j*8:])
	}
	ptrByte += len(pwd) / 8

	//Concatenates the salt
	for j := 0; j < len(salt)/8; j++ {
		wholeMatrix[ptrByte+j] = binary.LittleEndian.Uint64(salt[j*8:])
	}
	ptrByte += len(salt) / 8

	//Concatenates the basil: every integer passed as parameter, in the order they are provided by the interface
	wholeMatrix[ptrByte] = uint64(len(k))
	ptrByte++
	wholeMatrix[ptrByte] = uint64(len(pwd))
	ptrByte++
	wholeMatrix[ptrByte] = uint64(len(salt))
	ptrByte++
	wholeMatrix[ptrByte] = timeCost
	ptrByte++
	wholeMatrix[ptrByte] = uint64(nRows)
	ptrByte++
	wholeMatrix[ptrByte] = uint64(nCols)
	ptrByte++

	//Now comes the padding
	wholeMatrix[ptrByte] = 0x80 //first byte of padding: right after the password
	//resets the pointer to the start of the memory matrix
	ptrByte = (nBlocksInput*blockLenBlake2SafeBytes)/8 - 1 //sets the pointer to the correct position: end of incomplete block
	wholeMatrix[ptrByte] ^= 0x0100000000000000             //last byte of padding: at the end of the last incomplete block00
	//==========================================================================/

	//======================= Initializing the Sponge State ====================//
	//Sponge state: 16 uint64_t, BLOCK_LEN_INT64 words of them for the bitrate (b) and the remainder for the capacity (c)
	state := initState()
	//==========================================================================/

	//================================ Setup Phase =============================//
	//Absorbing salt, password and basil: this is the only place in which the block length is hard-coded to 512 bits
	ptrWord = 0
	for i = 0; i < nBlocksInput; i++ {
		absorbBlockBlake2Safe(state, wholeMatrix[ptrWord:]) //absorbs each block of pad(pwd || salt || basil)
		ptrWord += blockLenBlake2SafeInt64                  //goes to next block of pad(pwd || salt || basil)
	}

	//Initializes M[0] and M[1]
	reducedSqueezeRow0(state, memMatrix[0], nCols) //The locally copied password is most likely overwritten here
	reducedDuplexRow1(state, memMatrix[0], memMatrix[1], nCols)

	for row < nRows {
		//M[row] = rand; //M[row*] = M[row*] XOR rotW(rand)
		reducedDuplexRowSetup(state, memMatrix[prev], memMatrix[rowa], memMatrix[row], nCols)

		//updates the value of row* (deterministically picked during Setup))
		rowa = (rowa + uint64(step)) & (window - 1)
		//update prev: it now points to the last row ever computed
		prev = row
		//updates row: goes to the next row to be computed
		row++

		//Checks if all rows in the window where visited.
		if rowa == 0 {
			step = int(window + gap) //changes the step: approximately doubles its value
			window *= 2              //doubles the size of the re-visitation window
			gap = -gap               //inverts the modifier to the step
		}
	}
	//==========================================================================/

	//============================ Wandering Phase =============================//
	row = 0 //Resets the visitation to the first row of the memory matrix
	for tau = 1; tau <= timeCost; tau++ {
		//Step is approximately half the number of all rows of the memory matrix for an odd tau; otherwise, it is -1
		step = nRows/2 - 1
		if tau%2 == 0 {
			step = -1
		}

		for row0 := false; !row0; row0 = (row == 0) {
			//Selects a pseudorandom index row*
			//------------------------------------------------------------------------------------------
			//rowa = ((unsigned int)state[0]) & (nRows-1);	//(USE THIS IF nRows IS A POWER OF 2)
			rowa = state[0] % uint64(nRows) //(USE THIS FOR THE "GENERIC" CASE)
			//------------------------------------------------------------------------------------------

			//Performs a reduced-round duplexing operation over M[row*] XOR M[prev], updating both M[row*] and M[row]
			reducedDuplexRow(state, memMatrix[prev], memMatrix[rowa], memMatrix[row], nCols)

			//update prev: it now points to the last row ever computed
			prev = row

			//updates row: goes to the next row to be computed
			//------------------------------------------------------------------------------------------
			//row = (row + step) & (nRows-1);	//(USE THIS IF nRows IS A POWER OF 2)
			row = (row + step) % nRows //(USE THIS FOR THE "GENERIC" CASE)
			//------------------------------------------------------------------------------------------
		}
	}
	//==========================================================================/

	//============================ Wrap-up Phase ===============================//
	//Absorbs the last block of the memory matrix
	absorbBlock(state, memMatrix[rowa])
	//Squeezes the key
	squeeze(state, k)
	//==========================================================================/

}
