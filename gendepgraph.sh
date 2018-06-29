#!/bin/bash
godepgraph -s . | sed 's/github.com\/mit-dci\/lit\///' > dependency-graph.dot
