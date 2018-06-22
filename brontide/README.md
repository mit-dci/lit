brontide
==========

The brontide package implements a secure crypto messaging protocol based off of
the [Noise Protocol Framework](http://noiseprotocol.org/noise.html). The
package exposes the raw state machine that handles the handshake and subsequent
message encryption/decryption scheme. Additionally, the package exposes a
[net.Conn](https://golang.org/pkg/net/#Conn) and a
[net.Listener](https://golang.org/pkg/net/#Listener) interface implementation
which allows the encrypted transport to be seamlessly integrated into a
codebase.

The secure messaging scheme implemented within this package is described in
detail in [BOLT #8 of the Lightning Network specifications](https://github.com/lightningnetwork/lightning-rfc/blob/master/08-transport.md).

This package has intentionally been designed so it can be used as a standalone
package for any projects needing secure encrypted+authenticated communications
between network enabled programs.

This package requires additional attribution. Please see [license](LICENSE) for details.
