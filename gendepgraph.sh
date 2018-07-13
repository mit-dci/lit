#!/bin/bash
godepgraph -s . | sed 's/github.com\/mit-dci\/lit\///' | dot -Tpng -o docs/deps.png

