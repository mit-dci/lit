package lndc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net"
	"time"

	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
)

// Conn is an implementation of net.Conn which enforces an authenticated key
// exchange and message encryption protocol based off the noise_XX protocol
// In the case of a successful handshake, all
// messages sent via the .Write() method are encrypted with an AEAD cipher
// along with an encrypted length-prefix. See the Machine struct for
// additional details w.r.t to the handshake and encryption scheme.
type Conn struct {
	conn net.Conn

	noise *Machine

	readBuf bytes.Buffer
}

// A compile-time assertion to ensure that Conn meets the net.Conn interface.
var _ net.Conn = (*Conn)(nil)
var Noise_XK bool

// Dial attempts to establish an encrypted+authenticated connection with the
// remote peer located at address which has remotePub as its long-term static
// public key. In the case of a handshake failure, the connection is closed and
// a non-nil error is returned.
func Dial(localPriv *koblitz.PrivateKey, ipAddr string, remoteAddress string,
	dialer func(string, string) (net.Conn, error)) (*Conn, error) {
	var remotePKH string
	var remotePK [33]byte
	if remoteAddress[0:3] == "ln1" { // its a remote PKH
		remotePKH = remoteAddress
	} else if len(remoteAddress) == 66 { // hex encoded remotePK
		temp, _ := hex.DecodeString(remoteAddress)
		copy(remotePK[:], temp)
		logging.Info("Got remote PK: ", remotePK, ", using noise_xk to connect")
		SetXKConsts()
	}
	var conn net.Conn
	var err error
	conn, err = dialer("tcp", ipAddr)
	logging.Debug("ipAddr is: ", ipAddr)
	if err != nil {
		return nil, err
	}

	b := new(Conn)
	if Noise_XK {
		// we need to convert the raw PK to a koblitz public key
		remotePub, err := koblitz.ParsePubKey(remotePK[:], koblitz.S256())
		if err != nil {
			logging.Debug(err)
			return nil, err
		}
		b = &Conn{
			conn:  conn,
			noise: NewNoiseXKMachine(true, localPriv, remotePub),
		}
	} else {
		b = &Conn{
			conn:  conn,
			noise: NewNoiseXXMachine(true, localPriv),
		}
	}

	// Initiate the handshake by sending the first act to the receiver.
	actOne, err := b.noise.GenActOne(remotePK)
	if err != nil {
		b.conn.Close()
		return nil, err
	}
	if _, err := conn.Write(actOne[:]); err != nil {
		b.conn.Close()
		return nil, err
	}
	// We'll ensure that we get ActTwo from the remote peer in a timely
	// manner. If they don't respond within 1s, then we'll kill the
	// connection.
	conn.SetReadDeadline(time.Now().Add(handshakeReadTimeout))

	// If the first act was successful (we know that address is actually
	// remotePub), then read the second act after which we'll be able to
	// send our static public key to the remote peer with strong forward
	// secrecy.
	actTwo := make([]byte, ActTwoSize)
	if _, err := io.ReadFull(conn, actTwo[:]); err != nil {
		b.conn.Close()
		return nil, err
	}
	if !Noise_XK {
		remotePK, err = b.noise.RecvActTwo(actTwo)
		if err != nil {
			b.conn.Close()
			return nil, err
		}
	} else {
		if _, err := b.noise.RecvActTwo(actTwo); err != nil {
			b.conn.Close()
			return nil, err
		}
	}

	logging.Infoln("Received pubkey: ", remotePK)
	if lnutil.LitAdrFromPubkey(remotePK) != remotePKH && !Noise_XK {
		// for noise_XK dont check PKH and PK because we'd have already checked this
		// the last time we connected to this guy
		return nil, fmt.Errorf("Remote PKH doesn't match. Quitting!")
	}

	// Finally, complete the handshake by sending over our encrypted static
	// key and execute the final ECDH operation.
	actThree, err := b.noise.GenActThree()
	if err != nil {
		b.conn.Close()
		return nil, err
	}
	if _, err := conn.Write(actThree[:]); err != nil {
		b.conn.Close()
		return nil, err
	}

	// We'll reset the deadline as it's no longer critical beyond the
	// initial handshake.
	conn.SetReadDeadline(time.Time{})

	return b, nil
}

// ReadNextMessage uses the connection in a message-oriented instructing it to
// read the next _full_ message with the lndc stream. This function will
// block until the read succeeds.
func (c *Conn) ReadNextMessage() ([]byte, error) {
	return c.noise.ReadMessage(c.conn)
}

// Read reads data from the connection.  Read can be made to time out and
// return an Error with Timeout() == true after a fixed time limit; see
// SetDeadline and SetReadDeadline.
//
// Part of the net.Conn interface.
func (c *Conn) Read(b []byte) (n int, err error) {
	// In order to reconcile the differences between the record abstraction
	// of our AEAD connection, and the stream abstraction of TCP, we
	// maintain an intermediate read buffer. If this buffer becomes
	// depleted, then we read the next record, and feed it into the
	// buffer. Otherwise, we read directly from the buffer.
	if c.readBuf.Len() == 0 {
		plaintext, err := c.noise.ReadMessage(c.conn)
		if err != nil {
			return 0, err
		}

		if _, err := c.readBuf.Write(plaintext); err != nil {
			return 0, err
		}
	}

	return c.readBuf.Read(b)
}

// Write writes data to the connection.  Write can be made to time out and
// return an Error with Timeout() == true after a fixed time limit; see
// SetDeadline and SetWriteDeadline.
//
// Part of the net.Conn interface.
func (c *Conn) Write(b []byte) (n int, err error) {
	// If the message doesn't require any chunking, then we can go ahead
	// with a single write.
	if len(b) <= math.MaxUint16 {
		return len(b), c.noise.WriteMessage(c.conn, b)
	}

	// If we need to split the message into fragments, then we'll write
	// chunks which maximize usage of the available payload.
	chunkSize := math.MaxUint16

	bytesToWrite := len(b)
	bytesWritten := 0
	for bytesWritten < bytesToWrite {
		// If we're on the last chunk, then truncate the chunk size as
		// necessary to avoid an out-of-bounds array memory access.
		if bytesWritten+chunkSize > len(b) {
			chunkSize = len(b) - bytesWritten
		}

		// Slice off the next chunk to be written based on our running
		// counter and next chunk size.
		chunk := b[bytesWritten : bytesWritten+chunkSize]
		if err := c.noise.WriteMessage(c.conn, chunk); err != nil {
			return bytesWritten, err
		}

		bytesWritten += len(chunk)
	}

	return bytesWritten, nil
}

// Close closes the connection.  Any blocked Read or Write operations will be
// unblocked and return errors.
//
// Part of the net.Conn interface.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local network address.
//
// Part of the net.Conn interface.
func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
//
// Part of the net.Conn interface.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the
// connection. It is equivalent to calling both SetReadDeadline and
// SetWriteDeadline.
//
// Part of the net.Conn interface.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls.  A zero value for t
// means Read will not time out.
//
// Part of the net.Conn interface.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls.  Even if write
// times out, it may return n > 0, indicating that some of the data was
// successfully written.  A zero value for t means Write will not time out.
//
// Part of the net.Conn interface.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// RemotePub returns the remote peer's static public key.
func (c *Conn) RemotePub() *koblitz.PublicKey {
	return c.noise.remoteStatic
}

// LocalPub returns the local peer's static public key.
func (c *Conn) LocalPub() *koblitz.PublicKey {
	return c.noise.localStatic.PubKey()
}
