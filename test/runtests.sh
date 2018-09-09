#!/bin/bash

datadir=_data

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

for t in $tests; do

	if [ "$EXIT_REQED" == "1" ]; then
		ign=$(($ign + 1))
		continue
	fi

	echo "==============================="
	echo "Running test: $t"
	echo "==============================="
	ls

	echo "BEFORE THE TEST"
	python3 $PWD/itest_$t.py
	if [ $? != 0 ]; then
		echo "XXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
		echo "Failed: $t"
		echo "XXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
		fail=$(($fail + 1))
		continue
	fi

	echo "*******************************"
	echo "Compeleted: $t"
	echo "*******************************"
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
