package tor

import (
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

const (
	// base32Alphabet is the alphabet used for encoding and decoding v2 and
	// v3 onion addresses.
	base32Alphabet = "abcdefghijklmnopqrstuvwxyz234567"

	// OnionSuffix is the ".onion" suffix for v2 and v3 onion addresses.
	OnionSuffix = ".onion"

	// OnionSuffixLen is the length of the ".onion" suffix.
	OnionSuffixLen = len(OnionSuffix)

	// V2DecodedLen is the length of a decoded v2 onion service.
	V2DecodedLen = 10

	// V2Len is the length of a v2 onion service including the ".onion"
	// suffix.
	V2Len = 22

	// V3DecodedLen is the length of a decoded v3 onion service.
	V3DecodedLen = 35

	// V3Len is the length of a v2 onion service including the ".onion"
	// suffix.
	V3Len = 62

	// v2OnionAddr denotes a version 2 Tor onion service address.
	v2OnionAddr uint8 = 2

	// v3OnionAddr denotes a version 3 Tor (prop224) onion service address.
	v3OnionAddr uint8 = 3
)

var (
	// Base32Encoding represents the Tor's base32-encoding scheme for v2 and
	// v3 onion addresses.
	Base32Encoding = base32.NewEncoding(base32Alphabet)
)

// OnionAddr represents a Tor network end point onion address.
type OnionAddr struct {
	// OnionService is the host of the onion address.
	OnionService string

	// Port is the port of the onion address.
	Port int
}

// A compile-time check to ensure that OnionAddr implements the net.Addr
// interface.
var _ net.Addr = (*OnionAddr)(nil)

// String returns the string representation of an onion address.
func (o *OnionAddr) String() string {
	return net.JoinHostPort(o.OnionService, strconv.Itoa(o.Port))
}

// Network returns the network that this implementation of net.Addr will use.
// In this case, because Tor only allows TCP connections, the network is "tcp".
func (o *OnionAddr) Network() string {
	return "tcp"
}

// encodeOnionAddr serializes an onion address into its compact raw bytes
// representation.
func encodeOnionAddr(w io.Writer, addr *OnionAddr) error {
	var suffixIndex int
	switch len(addr.OnionService) {
	case V2Len:
		if _, err := w.Write([]byte{byte(v2OnionAddr)}); err != nil {
			return err
		}
		suffixIndex = V2Len - OnionSuffixLen
	case V3Len:
		if _, err := w.Write([]byte{byte(v3OnionAddr)}); err != nil {
			return err
		}
		suffixIndex = V3Len - OnionSuffixLen
	default:
		return errors.New("unknown onion service length")
	}

	host, err := Base32Encoding.DecodeString(
		addr.OnionService[:suffixIndex],
	)
	if err != nil {
		return err
	}
	if _, err := w.Write(host); err != nil {
		return err
	}

	var port [2]byte
	//byteOrder.PutUint16(port[:], uint16(addr.Port))
	if _, err := w.Write(port[:]); err != nil {
		return err
	}

	return nil
}

// deserializeAddr reads the serialized raw representation of an address and
// deserializes it into the actual address. This allows us to avoid address
// resolution within the channeldb package.
func DeserializeAddr(r io.Reader) (net.Addr, error) {
	var addrType [1]byte
	if _, err := r.Read(addrType[:]); err != nil {
		return nil, err
	}

	var address net.Addr
	switch uint8(addrType[0]) {
	case v2OnionAddr:
		var h [V2DecodedLen]byte
		if _, err := r.Read(h[:]); err != nil {
			return nil, err
		}

		var p [2]byte
		if _, err := r.Read(p[:]); err != nil {
			return nil, err
		}

		onionService := Base32Encoding.EncodeToString(h[:])
		onionService += OnionSuffix
		port := int(binary.BigEndian.Uint16(p[:]))

		address = &OnionAddr{
			OnionService: onionService,
			Port:         port,
		}
	case v3OnionAddr:
		var h [V3DecodedLen]byte
		if _, err := r.Read(h[:]); err != nil {
			return nil, err
		}

		var p [2]byte
		if _, err := r.Read(p[:]); err != nil {
			return nil, err
		}

		onionService := Base32Encoding.EncodeToString(h[:])
		onionService += OnionSuffix
		port := int(binary.BigEndian.Uint16(p[:]))

		address = &OnionAddr{
			OnionService: onionService,
			Port:         port,
		}
	default:
		//return nil, ErrUnknownAddressType
		return nil, fmt.Errorf("Unknown address type")
	}

	return address, nil
}

// serializeAddr serializes an address into its raw bytes representation so that
// it can be deserialized without requiring address resolution.
func SerializeAddr(w io.Writer, address net.Addr) error {
	switch addr := address.(type) {
	case *OnionAddr:
		return encodeOnionAddr(w, addr)
	}

	return nil
}
