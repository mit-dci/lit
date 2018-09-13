#!/bin/bash +e
set +e
tests=$(cat tests.txt | grep -vE '^(#|$)' | sed 's/ *#.*//g')
if [ "$#" -gt 0 ]; then
	tests=$@
fi

echo 'Tests to run:'
for t in $tests; do
	echo "* $t"
done
echo '************************************************'

chmod +x run_itests.py
./run_itests.py
