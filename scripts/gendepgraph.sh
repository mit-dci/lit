#!/bin/bash

out=docs/deps.png

if [ -n "$1" ]; then
	out=$1
fi

godepgraph -s . | sed 's/github.com\/mit-dci\/lit\///' | dot -Tpng -o $1
