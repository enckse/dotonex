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

echo "==========================="
echo "running $CONF"
echo "==========================="
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
pkill --signal 2 radiucal
sleep 1
pkill radiucal
pkill harness

COMPARE="results stats logger access"
cat ${LOGS}audit* | cut -d " " -f 2- > bin/results.log
rm -f bin/stats.log
rm -f bin/logger.log
rm -f bin/access.log

for f in $(echo "acct.stats.accounting stats.trace stats.preauth stats.postauth"); do
    cat ${LOGS}${f}.* | grep -v -E "^(first|last)" >> bin/stats.log
done

_catlogs() {
    for f in $(ls $LOGS | grep "auxiliary" | sort); do
        cat ${LOGS}$f | grep "[$1]" | cut -d " " -f 3- | \
            sed "s/^  //g" | cut -d " " -f 1,3 | \
            sed "s/^Access/ Access/g" | \
            sed "s/^UDPAddr/ UDPAddr/g" | \
            sed "s/^Id/ Id/g" | \
            cut -d " " -f 1,2 
    done
}

for o in access logger; do
    _catlogs $o | sort >> bin/$o.log
done

for d in $(echo $COMPARE); do
    diff --color -u tests/expected.$CONF.$d.log bin/$d.log
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

reject=8
if [[ "$CONF" == "norjct" ]]; then
    reject=0
fi

_checks "rejecting client" $reject
_checks "client failed auth check" 8
echo "stdout checks passed"
sleep 3
echo "$CONF is completed..."
