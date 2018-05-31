package lndc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// maxMsgSize is the longest message supported.  This is payload size, not
// encrypted size.
const maxMsgSize = 1 << 24

// New & improved tcp open session.
// There's connector A and listener B.  Once the connection is set up there's no
// difference, but there can be during the setup.
// Setup:
// 1 -> A sends B ephemeral secp256k1 pubkey (33 bytes)
// 2 <- B sends A ephemeral secp256k1 pubkey (33 bytes)
// A and B do DH, get a shared secret.
// ==========
// Seesion is open!  Done!  Well not quite.  Session is confidential but not
// yet authenticated.  From here on, can use the Send() and Recv() functions with
// chacha20poly1305.
// ==========

// Nodes authenticate by doing a DH with their persistent identity keys, and then
// exchanging hash based proofs that they got the same shared IDDH secret.
// The DH proof is h160(remote eph pubkey, IDDH secret)
// A initiates auth.
//
// If A does not know B's pubkey but only B's pubkey hash:
//
// 1 -> A sends [PubKeyA, PubKeyHashB] (53 bytes)
// B computes ID pubkey DH
// 2 <- B sends [PubkeyB, DH proof] (53 bytes)
// 3 -> A sends DH proof (20 bytes)
// done.
//
// This exchange can be sped up if A already knows B's pubkey:
//
// A already knows who they're talking to, or trying to talk to
// 1 -> A sends [PubKeyA, PubkeyHashB, DH proof] (73 bytes)
// 2 <- B sends DH proof (20 bytes)
//
// A and B both verify those H160 hashes, and if matching consider their
// session counterparty authenticated.
//
// A possible weakness of the DH proof is if B re-uses eph keys.  That potentially
// makes *A*'s proof weaker though.  A gets to choose the proof B creates.  As
// long as your software makes new eph keys each time, you should be OK.

// readClear and writeClear don't encrypt but directly read and write to the
// underlying data link, only adding or subtracting a 2 byte length header.
// All Read() and Write() calls for lndc's use these functions internally
// (they aren't exported).  They're also used in the key agreement phase.

// readClear reads the next length-prefixed message from the underlying raw
// TCP connection.
func readClear(c net.Conn) ([]byte, error) {
	var msgLen uint32

	if err := binary.Read(c, binary.BigEndian, &msgLen); err != nil {
		return nil, err
	}

	msg := make([]byte, msgLen)
	if _, err := io.ReadFull(c, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// writeClear writes the passed message with a prefixed 2-byte length header.
func writeClear(conn net.Conn, msg []byte) (int, error) {
	if len(msg) > maxMsgSize {
		return 0, fmt.Errorf("lmsg too long, %d bytes", len(msg))
	}

	// Add 2 byte length header (pbx doesn't need it) and send over TCP.
	var msgBuf bytes.Buffer
	if err := binary.Write(&msgBuf, binary.BigEndian, uint32(len(msg))); err != nil {
		return 0, err
	}

	if _, err := msgBuf.Write(msg); err != nil {
		return 0, err
	}

	return conn.Write(msgBuf.Bytes())
}
