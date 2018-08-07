#!/bin/sh

# This file is temporary for use during development.  Printing to stderr while
# the UI is up causes problems, so we redirect it.

cargo build
cargo run $1 $2 2> stderr.log
