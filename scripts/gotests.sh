#!/bin/bash

scriptpath=$(realpath $0)

for td in */; do
	if [ "$td" == "build/" ]; then
		continue
	fi
	if [ "$td" == "cmd/" ]; then
		continue
	fi

	gocnt=$(find $td -name '*.go' | wc -l)
	if [ $gocnt -gt 0 ]; then
		echo "Running go test in $td"
		$(dirname $(dirname $scriptpath))'/build/env.sh' go test -v ./$td
		echo ''
	fi
done
