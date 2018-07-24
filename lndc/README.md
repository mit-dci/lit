lndc
==========

The lndc package implements a secure crypto messaging protocol based off of
the [Noise Protocol Framework](http://noiseprotocol.org/noise.html). The
package exposes the raw state machine that handles the handshake and subsequent
message encryption/decryption scheme. Additionally, the package exposes a
[net.Conn](https://golang.org/pkg/net/#Conn) and a
[net.Listener](https://golang.org/pkg/net/#Listener) interface implementation
which allows the encrypted transport to be seamlessly integrated into a
codebase.

The secure messaging scheme implemented within this package uses `NOISE_XX` as the handshake for authenticated key exchange. Please note that this is not the same as [brontide](https://github.com/lightningnetwork/lnd/tree/master/brontide) which uses the `NOISE_XK` protocol for handshakes and `lndc` is not compatible with the same.

This package has intentionally been designed so it can be used as a standalone
package for any projects needing secure encrypted+authenticated communications
between network enabled programs.

This package requires additional attribution to that of lit since it is adapted from the original [brontide](https://github.com/lightningnetwork/lnd/tree/master/brontide) package. Please see [license](LICENSE) for details.
