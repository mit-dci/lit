#!/bin/bash +e

datadir=_data
set +e
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

export EXIT_REQED=0
export TEST_PID=-1

./test_connect.py
for t in $tests; do
	echo "==============================="
	echo "Running test: $t"
	echo "==============================="
	ls

	echo "BEFORE THE TEST"
	{
		./itest_$t.py
	} || {
		echo "XXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
		echo "Failed: $t"
		echo "XXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
		break
	}
	echo "*******************************"
	echo "Completed: $t"
	echo "*******************************"
	echo "COOL, SOMETHING WORKS"
	exec ls
	exec rm -rf _data/
done

if [ "$EXIT_REQED" == "1" ]; then
	kill -2 $TEST_PID
	echo 'Waiting for tests to exit... (This may take several seconds)'
	wait $TEST_PID
fi

echo 'All tests completed.'
echo "Passed: $n"
