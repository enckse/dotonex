#!/bin/bash
OUT=bin/stdout
LOGS=tests/log/
CONF="$1"
_run() {
    bin/radiucal --config tests/test.$CONF.conf > $OUT 2>&1
}

_acct() {
    bin/radiucal --instance acct --config tests/test.acct.conf > $OUT.acct 2>&1
}

_run &
_acct &
sleep 1
echo "starting harness..."
bin/harness --endpoint=true &
sleep 1
echo "running tests..."
bin/harness
echo "reloading..."
kill -2 $(pidof radiucal)
echo "re-running..."
bin/harness
sleep 1
echo "killing..."
pkill radiucal
pkill harness

COMPARE="results stats"
cat ${LOGS}radiucal.audit* | cut -d " " -f 2- > bin/results.log
rm -f bin/stats.log
for f in $(echo "acct.stats.accounting stats.trace stats.preauth stats.postauth"); do
    cat ${LOGS}/radiucal.${f}.* | grep -v -E "^(first|last)" >> bin/stats.log
done

for d in $(echo $COMPARE); do
    diff -u bin/$d.log tests/expected.$CONF.$d.log
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
_checks() {
    cnt=$(cat $OUT | grep "$1" | wc -l)
    if [ $cnt -ne $2 ]; then
        echo "invalid stdout log: $1 ($cnt)"
        exit 1
    fi
}

reject=2
if [[ "$CONF" == "norjct" ]]; then
    reject=0
fi

_checks "rejecting client" $reject
_checks "client failed preauth" 2
echo "stdout checks passed"
sleep 3
echo "$CONF is completed..."
