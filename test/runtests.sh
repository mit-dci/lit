#!/bin/bash

datadir=_data

set -x

tests=$(cat tests.txt | grep -vE '^(#|$)' | sed 's/ *#.*//g')
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
fail=0
ign=0

export EXIT_REQED=0
export TEST_PID=-1

function killcurrentandexit () {
	echo 'testpid!' $TEST_PID
	export EXIT_REQED=1
}

trap killcurrentandexit INT

echo 'Umask:' $(umask -S)

for t in $tests; do

	if [ "$EXIT_REQED" == "1" ]; then
		ign=$(($ign + 1))
		continue
	fi

	echo "Running test: $t"
	echo '===='

	tdata=$datadir/$t

	if [ -e "$tdata" ]; then
		echo 'Directory found, was this test already run?'
		echo '----'
		ign=$(($ign + 1))
		continue
	fi

	ls -l . $tdata
	mkdir -p $tdata
	env LIT_ITEST_ROOT=$(realpath $tdata) ./itest_$t.py & export TEST_PID=$!
	wait $TEST_PID
	if [ $? != 0 ]; then
		echo -e "\n====\n"
		echo "Failed: $t"
		echo '---'
		fail=$(($fail + 1))
		continue
	fi

	echo -e "\n===="
	echo "Compeleted: $t"
	echo '----'
	n=$(($n + 1))

done

if [ "$EXIT_REQED" == "1" ]; then
	kill -2 $TEST_PID
	echo 'Waiting for tests to exit... (This may take several seconds)'
	wait $TEST_PID
fi

echo 'All tests completed.'
echo "Passed: $n"
echo "Failed: $fail"
echo "Ignored: $ign"

if [ "$fail" -ne "0" ]; then
	exit 1
fi
