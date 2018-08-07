#!/bin/bash

datadir=_data

tests=$(cat tests.txt | grep -vE '^(#|$)')

echo 'Tests to run:'
for t in $tests; do
	echo "* $t"
done
echo '========'

n=0

for t in $tests; do
	echo "Running test: $t"
	echo '===='

	tdata=$datadir/$t
	mkdir -p $tdata
	env LIT_ITEST_ROOT=$(realpath $tdata) ./itest_$t.py
	if [ $? != 0 ]; then
		echo -e "\n====\n"
		echo "Failed: $t"
		exit 1
	fi

	echo -e "\n===="
	echo "Compeleted: $t"
	echo '----'
	n=$(($n + 1))
done

rm -rf $datadir

echo "All ($n) tests passed."
