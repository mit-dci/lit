[![Godoc Reference](https://godoc.org/github.com/aead/skein?status.svg)](https://godoc.org/github.com/aead/skein)

## The Skein hash function family

Skein is a cryptographic hash function family designed by  Bruce Schneier, Niels Ferguson, Stefan Lucks,
Doug Whiting, Mihir Bellare, Tadayoshi Kohno, Jon Callas and Jesse Walker.

Skein uses the tweakable block cipher Threefish in UBI chaining mode and can produce hash values of any length.
There exists three Skein variants:
 - Skein-256 based on Threefish-256 with a block size of 256 bit.
 - Skein-512 based on Threefish-512 with a block size of 512 bit. (This variant is recommended)
 - Skein-1024 based on Threefish-1024 with a block size of 1024 bit. (Very conservative security level)

### Installation
Install in your GOPATH: `go get -u github.com/aead/skein`  
