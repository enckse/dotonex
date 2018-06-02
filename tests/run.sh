#!/bin/bash
bin/radiucal --config tests/test.conf &
bin/radiucal --instance acct --config tests/test.acct.conf &
sleep 1
bin/harness --endpoint=true &
sleep 1
bin/harness
kill -2 $(pidof radiucal)
bin/harness
sleep 1
pkill radiucal
pkill harness

COMPARE="results stats"
cat tests/log/radiucal.audit* | cut -d " " -f 2- > bin/results.log
rm -f bin/stats.log
for f in $(echo "acct.stats.accounting stats.trace stats.preauth"); do
    cat tests/log/radiucal.${f}.* | grep -v -E "^(first|last)" >> bin/stats.log
done

for d in $(echo $COMPARE); do
    diff -u bin/$d.log tests/expected.$d.log
    if [ $? -ne 0 ]; then
        echo "integration test failed ($d)"
        exit 1
    fi
done

echo "logged results match"
if cat bin/count | grep -q "^count:4$"; then
    echo "count passes"
else
    echo "invalid count"
    exit 1
fi
