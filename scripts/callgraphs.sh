#!/bin/bash

pkgs='qln lnutil lndc litrpc powless uspv wallit watchtower dlc elkrem portxo coinparam sig64 btcutil wire'

echo "Generating call graphs for packages: $pkgs"

outdir=build/graphs
mkdir -p $outdir

for p in $pkgs; do
	graphpath=$outdir/callgraph-$p.dot
	pngpath=$outdir/callgraph-$p.png
	echo 'graphing:' $p
	go-callvis \
		-group pkg,type \
		-nostd \
		-focus $p \
		-minlen 12 \
		. > $graphpath
	echo 'rendering:' $p
	dot -Tpng $graphpath -o $pngpath
done
