#!/bin/bash +e

tests=$(cat tests.txt | grep -vE '^(#|$)' | sed 's/ *#.*//g')
if [ "$#" -gt 0 ]; then
	tests=$@
fi

echo 'Tests to run:'
for t in $tests; do
	echo "* $t"
done
echo '************************'

./itest_connect.py
./itest_receive.py
./itest_send.py
./itest_send2.py
./itest_setgetfee.py
./itest_fund.py
./itest_close.py
./itest_close_reverse.py
./itest_break.py
./itest_break_reverse.py
./itest_push.py
./itest_pushclose.py
./itest_pushclose_reverse.py
./itest_pushbreak.py
./itest_pushbreak_reverse.py
