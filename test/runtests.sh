#!/bin/bash

datadir=_data

tests=$(cat tests.txt | grep -vE '^(#|$)')
if [ "$#" -gt 0 ]; then
	tests=$@
fi

echo 'Tests to run:'
for t in $tests; do
	echo "* $t"
done
echo '========'

rm -rf $datadir

n=0
ign=0

for t in $tests; do
	echo "Running test: $t"
	echo '===='

	tdata=$datadir/$t

	if [ -e "$tdata" ]; then
		echo 'Directory found, was this test already run?'
		echo '----'
		ign=$(($ign + 1))
		continue
	fi

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

echo "All ($n) tests passed.  (Ignored: $ign)"
