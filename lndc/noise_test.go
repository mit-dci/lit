package lndc

import (
	"bytes"
	"encoding/hex"
	"io"
	"log"
	"math"
	"net"
	"sync"
	"testing"

	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/lnutil"
)

type maybeNetConn struct {
	conn net.Conn
	err  error
}

func makeListener() (*Listener, string, string, error) {
	// First, generate the long-term private keys for the lndc listener.
	localPriv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, "", "", err
	}

	// Having a port of ":0" means a random port, and interface will be
	// chosen for our listener.
	addr := "localhost:0"

	// Our listener will be local, and the connection remote.
	listener, err := NewListener(localPriv, addr)
	if err != nil {
		return nil, "", "", err
	}
	var idPub [33]byte
	copy(idPub[:], localPriv.PubKey().SerializeCompressed())
	lisAdr := lnutil.LitAdrFromPubkey(idPub)
	return listener, lisAdr, listener.Addr().String(), nil
}

func establishTestConnection(wrong bool) (net.Conn, net.Conn, func(), error) {
	listener, pkh, netAddr, err := makeListener()
	if err != nil {
		return nil, nil, nil, err
	}
	defer listener.Close()
	// Nos, generate the long-term private keys remote end of the connection
	// within our test.
	remotePriv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, nil, err
	}

	// Initiate a connection with a separate goroutine, and listen with our
	// main one. If both errors are nil, then encryption+auth was
	// successful.
	if wrong {
		pkh = "ln1p7lhcxmlfgd5mltv6pc335aulv443tkw49q6er"
		log.Println("Trying to connect to wrong pk hash:", pkh)
	}
	remoteConnChan := make(chan maybeNetConn, 1)
	go func() {
		remoteConn, err := Dial(remotePriv, netAddr, pkh, net.Dial)
		if err != nil {
			log.Println(err)
		}
		remoteConnChan <- maybeNetConn{remoteConn, err}
	}()

	localConnChan := make(chan maybeNetConn, 1)
	go func() {
		localConn, err := listener.Accept()
		localConnChan <- maybeNetConn{localConn, err}
	}()

	remote := <-remoteConnChan
	if remote.err != nil {
		return nil, nil, nil, remote.err
	}

	local := <-localConnChan
	if local.err != nil {
		return nil, nil, nil, local.err
	}

	cleanUp := func() {
		local.conn.Close()
		remote.conn.Close()
	}
	return local.conn, remote.conn, cleanUp, nil
}


func TestConnectionCorrectness(t *testing.T) {
	// Create a test connection, grabbing either side of the connection
	// into local variables. If the initial crypto handshake fails, then
	// we'll get a non-nil error here.
	_, _, _, err := establishTestConnection(true) // wrong pkh
	if err == nil {
		t.Fatalf("Failed to catch bad connection: %v", err)
	}
	localConn, remoteConn, cleanUp, err := establishTestConnection(false) // correct pkh
	if err != nil {
		t.Fatalf("unable to establish test connection: %v", err)
	}
	defer cleanUp()

	// Test out some message full-message reads.
	for i := 0; i < 10; i++ {
		msg := []byte("hello" + string(i))

		if _, err := localConn.Write(msg); err != nil {
			t.Fatalf("remote conn failed to write: %v", err)
		}

		readBuf := make([]byte, len(msg))
		if _, err := remoteConn.Read(readBuf); err != nil {
			t.Fatalf("local conn failed to read: %v", err)
		}

		if !bytes.Equal(readBuf, msg) {
			t.Fatalf("messages don't match, %v vs %v",
				string(readBuf), string(msg))
		}
	}

	// Now try incremental message reads. This simulates first writing a
	// message header, then a message body.
	outMsg := []byte("hello world")
	if _, err := localConn.Write(outMsg); err != nil {
		t.Fatalf("remote conn failed to write: %v", err)
	}

	readBuf := make([]byte, len(outMsg))
	if _, err := remoteConn.Read(readBuf[:len(outMsg)/2]); err != nil {
		t.Fatalf("local conn failed to read: %v", err)
	}
	if _, err := remoteConn.Read(readBuf[len(outMsg)/2:]); err != nil {
		t.Fatalf("local conn failed to read: %v", err)
	}

	if !bytes.Equal(outMsg, readBuf) {
		t.Fatalf("messages don't match, %v vs %v",
			string(readBuf), string(outMsg))
	}
}

// TestConecurrentHandshakes verifies the listener's ability to not be blocked
// by other pending handshakes. This is tested by opening multiple tcp
// connections with the listener, without completing any of the noise_XX acts.
// The test passes if real lndc dialer connects while the others are
// stalled.
func TestConcurrentHandshakes(t *testing.T) {
	listener, pubKey, netAddr, err := makeListener()
	if err != nil {
		t.Fatalf("unable to create listener connection: %v", err)
	}
	defer listener.Close()

	const nblocking = 5

	// Open a handful of tcp connections, that do not complete any steps of
	// the noise_XX handshake.
	connChan := make(chan maybeNetConn)
	for i := 0; i < nblocking; i++ {
		go func() {
			conn, err := net.Dial("tcp", listener.Addr().String())
			connChan <- maybeNetConn{conn, err}
		}()
	}

	// Receive all connections/errors from our blocking tcp dials. We make a
	// pass to gather all connections and errors to make sure we defer the
	// calls to Close() on all successful connections.
	tcpErrs := make([]error, 0, nblocking)
	for i := 0; i < nblocking; i++ {
		result := <-connChan
		if result.conn != nil {
			defer result.conn.Close()
		}
		if result.err != nil {
			tcpErrs = append(tcpErrs, result.err)
		}
	}
	for _, tcpErr := range tcpErrs {
		if tcpErr != nil {
			t.Fatalf("unable to tcp dial listener: %v", tcpErr)
		}
	}

	// Now, construct a new private key and use the lndc dialer to
	// connect to the listener.
	remotePriv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Fatalf("unable to generate private key: %v", err)
	}

	go func() {
		remoteConn, err := Dial(remotePriv, netAddr, pubKey, net.Dial)
		connChan <- maybeNetConn{remoteConn, err}
	}()

	// This connection should be accepted without error, as the lndc
	// connection should bypass stalled tcp connections.
	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("unable to accept dial: %v", err)
	}
	defer conn.Close()

	result := <-connChan
	if result.err != nil {
		t.Fatalf("unable to dial %v: %v", netAddr, result.err)
	}
	result.conn.Close()
}

func TestMaxPayloadLength(t *testing.T) {
	t.Parallel()

	b := Machine{}
	b.split()

	var buf bytes.Buffer
	// Generate another payload which should be accepted as a valid
	// payload.
	payloadToAccept := make([]byte, math.MaxUint16-1)
	payloadToReject := make([]byte, math.MaxUint16+1)
	if b.WriteMessage(&buf, payloadToAccept) != nil || b.WriteMessage(&buf, payloadToReject) == nil {
		t.Fatalf("write for payload was rejected, should have been " +
			"accepted")
	}
}

func TestWriteMessageChunking(t *testing.T) {
	// Create a test connection, grabbing either side of the connection
	// into local variables. If the initial crypto handshake fails, then
	// we'll get a non-nil error here.
	localConn, remoteConn, cleanUp, err := establishTestConnection(false)
	if err != nil {
		t.Fatalf("unable to establish test connection: %v", err)
	}
	defer cleanUp()

	// Attempt to write a message which is over 3x the max allowed payload
	// size.
	largeMessage := bytes.Repeat([]byte("kek"), math.MaxUint16*3)

	// Launch a new goroutine to write the large message generated above in
	// chunks. We spawn a new goroutine because otherwise, we may block as
	// the kernel waits for the buffer to flush.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		bytesWritten, err := localConn.Write(largeMessage)
		if err != nil {
			t.Fatalf("unable to write message: %v", err)
		}

		// The entire message should have been written out to the remote
		// connection.
		if bytesWritten != len(largeMessage) {
			t.Fatalf("bytes not fully written!")
		}

		wg.Done()
	}()

	// Attempt to read the entirety of the message generated above.
	buf := make([]byte, len(largeMessage))
	if _, err := io.ReadFull(remoteConn, buf); err != nil {
		t.Fatalf("unable to read message: %v", err)
	}

	wg.Wait()

	// Finally, the message the remote end of the connection received
	// should be identical to what we sent from the local connection.
	if !bytes.Equal(buf, largeMessage) {
		t.Fatalf("bytes don't match")
	}
}

func TestBolt0008TestVectors(t *testing.T) {
	t.Parallel()

	// First, we'll generate the state of the initiator from the test
	// vectors at the appendix of BOLT-0008
	initiatorKeyBytes, err := hex.DecodeString("1111111111111111111111" +
		"111111111111111111111111111111111111111111")
	if err != nil {
		t.Fatalf("unable to decode hex: %v", err)
	}
	initiatorPriv, _ := btcec.PrivKeyFromBytes(btcec.S256(),
		initiatorKeyBytes)

	// We'll then do the same for the responder.
	responderKeyBytes, err := hex.DecodeString("212121212121212121212121" +
		"2121212121212121212121212121212121212121")
	if err != nil {
		t.Fatalf("unable to decode hex: %v", err)
	}
	responderPriv, _ := btcec.PrivKeyFromBytes(btcec.S256(),
		responderKeyBytes)

	// With the initiator's key data parsed, we'll now define a custom
	// EphemeralGenerator function for the state machine to ensure that the
	// initiator and responder both generate the ephemeral public key
	// defined within the test vectors.
	initiatorEphemeral := EphemeralGenerator(func() (*btcec.PrivateKey, error) {
		e := "121212121212121212121212121212121212121212121212121212" +
			"1212121212"
		eBytes, err := hex.DecodeString(e)
		if err != nil {
			return nil, err
		}

		priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), eBytes)
		return priv, nil
	})
	responderEphemeral := EphemeralGenerator(func() (*btcec.PrivateKey, error) {
		e := "222222222222222222222222222222222222222222222222222" +
			"2222222222222"
		eBytes, err := hex.DecodeString(e)
		if err != nil {
			return nil, err
		}

		priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), eBytes)
		return priv, nil
	})

	// Finally, we'll create both brontide state machines, so we can begin
	// our test.
	initiator := NewNoiseMachine(true, initiatorPriv, initiatorEphemeral)
	responder := NewNoiseMachine(false, responderPriv, responderEphemeral)

	// We'll start with the initiator generating the initial payload for
	// act one. This should consist of exactly 50 bytes. We'll assert that
	// the payload return is _exactly_ the same as what's specified within
	// the test vectors.
	actOne, err := initiator.GenActOne()
	if err != nil {
		t.Fatalf("unable to generate act one: %v", err)
	}
	expectedActOne, err := hex.DecodeString("01036360e856310ce5d294e" +
		"8be33fc807077dc56ac80d95d9cd4ddbd21325eff73f71432d5611e91" +
		"ffea67c17e8d5ae0cbb3")
	if err != nil {
		t.Fatalf("unable to parse expected act one: %v", err)
	}
	if !bytes.Equal(expectedActOne, actOne[:]) {
		t.Fatalf("act one mismatch: expected %x, got %x",
			expectedActOne, actOne)
	}

	// With the assertion above passed, we'll now process the act one
	// payload with the responder of the crypto handshake.
	if err := responder.RecvActOne(actOne); err != nil {
		t.Fatalf("responder unable to process act one: %v", err)
	}

	// Next, we'll start the second act by having the responder generate
	// its contribution to the crypto handshake. We'll also verify that we
	// produce the _exact_ same byte stream as advertised within the spec's
	// test vectors.
	actTwo, err := responder.GenActTwo()
	if err != nil {
		t.Fatalf("unable to generate act two: %v", err)
	}
	expectedActTwo, err := hex.DecodeString("0102466d7fcae563e5cb09a0" +
		"d1870bb580344804617879a14949cf22285f1bae3f27028d7500dd4c126" +
		"85d1f568b4c2b5048e8534b873319f3a8daa612b469132ec7f724fb90ec" +
		"6cbfad43030deee7f279410b")
	if err != nil {
		t.Fatalf("unable to parse expected act two: %v", err)
	}
	if !bytes.Equal(expectedActTwo, actTwo[:]) {
		t.Fatalf("act two mismatch: expected %x, got %x",
			expectedActTwo, actTwo)
	}

	// Moving the handshake along, we'll also ensure that the initiator
	// accepts the act two payload.
	if _, err := initiator.RecvActTwo(actTwo); err != nil {
		t.Fatalf("initiator unable to process act two: %v", err)
	}

	// At the final step, we'll generate the last act from the initiator
	// and once again verify that it properly matches the test vectors.
	actThree, err := initiator.GenActThree()
	if err != nil {
		t.Fatalf("unable to generate act three: %v", err)
	}
	expectedActThree, err := hex.DecodeString("018ac8fc232a47aa6fa5c51" +
		"b3b72c5824018e9d92f0840a5eada20f3b00d66a0e4c93b4e638aad3" +
		"6083982b74ae15f25f21aca63afa221bc26ea734ca44e8d01aa7e")
	if err != nil {
		t.Fatalf("unable to parse expected act three: %v", err)
	}
	if !bytes.Equal(expectedActThree, actThree[:]) {
		t.Fatalf("act three mismatch: expected %x, got %x",
			expectedActThree, actThree)
	}

	// Finally, we'll ensure that the responder itself also properly parses
	// the last payload in the crypto handshake.
	if err := responder.RecvActThree(actThree); err != nil {
		t.Fatalf("responder unable to process act three: %v", err)
	}

	// As a final assertion, we'll ensure that both sides have derived the
	// proper symmetric encryption keys.
	sendingKey, err := hex.DecodeString("6645a2f8c64cc44d0b95614cbe51c2c9c" +
	"1bee9945bfee823120b5a0978424bdf")
	if err != nil {
		t.Fatalf("unable to parse sending key: %v", err)
	}
	recvKey, err := hex.DecodeString("43b4a250b7b71ec303fb28b702b85a634" +
	"9fd9849662e8de3e5cee770f499e449")
	if err != nil {
		t.Fatalf("unable to parse receiving key: %v", err)
	}

	chainKey, err := hex.DecodeString("7e3044d33f4184f65c836133206576b49" +
	"a9c1cde623321afdcbb39624af60a99")
	if err != nil {
		t.Fatalf("unable to parse chaining key: %v", err)
	}

	if !bytes.Equal(initiator.sendCipher.secretKey[:], sendingKey) {
		t.Fatalf("sending key mismatch: expected %x, got %x",
			initiator.sendCipher.secretKey[:], sendingKey)
	}
	if !bytes.Equal(initiator.recvCipher.secretKey[:], recvKey) {
		t.Fatalf("receiving key mismatch: expected %x, got %x",
			initiator.recvCipher.secretKey[:], recvKey)
	}
	if !bytes.Equal(initiator.chainingKey[:], chainKey) {
		t.Fatalf("chaining key mismatch: expected %x, got %x",
			initiator.chainingKey[:], chainKey)
	}

	if !bytes.Equal(responder.sendCipher.secretKey[:], recvKey) {
		t.Fatalf("sending key mismatch: expected %x, got %x",
			responder.sendCipher.secretKey[:], recvKey)
	}
	if !bytes.Equal(responder.recvCipher.secretKey[:], sendingKey) {
		t.Fatalf("receiving key mismatch: expected %x, got %x",
			responder.recvCipher.secretKey[:], sendingKey)
	}
	if !bytes.Equal(responder.chainingKey[:], chainKey) {
		t.Fatalf("chaining key mismatch: expected %x, got %x",
			responder.chainingKey[:], chainKey)
	}

	// Now test as per section "transport-message test" in Test Vectors
	// (the transportMessageVectors ciphertexts are from this section of BOLT 8);
	// we do slightly greater than 1000 encryption/decryption operations
	// to ensure that the key rotation algorithm is operating as expected.
	// The starting point for enc/decr is already guaranteed correct from the
	// above tests of sendingKey, receivingKey, chainingKey.
	transportMessageVectors := map[int]string{
		0: "78fcfa42dcbf9f174abaea90dec3a678cc26a15700d8aaf7e5395a187e3" +
			"a1ab176e7cb1ec33a66",
		1: "c840d0ba1869e362d609815b68d0adbf6213b14f846cb1369e39352562e" +
			"58403e782f7ffacefd6",
		500: "9e3be84dae80d3900f50bd29a265fdf9c6745042e6054c7d84a2a81a4" +
			"4ddea9108dc3411c07ea8",
		501: "1ae1c6f783bfada390f7f1edb50ab0c48c0d5effb679610299fdf3b8c" +
			"1d3c0b14656fa2692ff8e",
		1000: "0a8cbec4586154871b8bf04f8efa97b183244ed2b269796c319bf0c4" +
			"78f3cdeeef11e8a86ce9fd",
		1001: "2a48d153ab9f01328a276c2f132ba67dd6a9b629899787eea2a402159" +
			"cbb85aa22a4dff2071042",
	}

	// Payload for every message is the string "hello".
	payload := []byte("hello")

	var buf bytes.Buffer

	for i := 0; i < 1002; i++ {
		err = initiator.WriteMessage(&buf, payload)
		if err != nil {
			t.Fatalf("could not write message %s", payload)
		}
		if val, ok := transportMessageVectors[i]; ok {
			binaryVal, err := hex.DecodeString(val)
			if err != nil {
				t.Fatalf("Failed to decode hex string %s", val)
			}
			if !bytes.Equal(buf.Bytes(), binaryVal) {
				t.Fatalf("Ciphertext %x was not equal to expected %s",
					buf.String()[:], val)
			}
		}

		// Responder decrypts the bytes, in every iteration, and
		// should always be able to decrypt the same payload message.
		plaintext, err := responder.ReadMessage(&buf)
		if err != nil {
			t.Fatalf("failed to read message in responder: %v", err)
		}

		// Ensure decryption succeeded
		if !bytes.Equal(plaintext, payload) {
			t.Fatalf("Decryption failed to receive plaintext: %s, got %s",
				payload, plaintext)
		}

		// Clear out the buffer for the next iteration
		buf.Reset()
	}
}
