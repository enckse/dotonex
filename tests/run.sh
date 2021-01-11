#!/bin/bash
BIN=bin/
rm -rf $BIN
mkdir -p $BIN
OUT=${BIN}stdout
LOGS=log/
PLUGINS=plugins/
HARNESS="../tools/harness.go"
for d in $LOGS $PLUGINS; do
    rm -rf $d
    mkdir -p $d
done

PATH="../:$PATH"
CONF="$1"
_run() {
    ../radiucal --debug --config test.$CONF.conf > $OUT 2>&1
}

_acct() {
    ../radiucal --debug --instance acct --config test.acct.conf > $OUT.acct 2>&1
}

_reset() {
    pkill --signal 2 radiucal-runner
}

echo "==========================="
echo "running $CONF (from: $PWD)"
echo "==========================="
_run &
_acct &
sleep 1
echo "starting harness..."
go run $HARNESS --endpoint=true &
sleep 1
echo "running tests..."
go run $HARNESS
echo "reloading..."
_reset
echo "re-running..."
go run $HARNESS
sleep 1
echo "killing..."
_reset
sleep 1
pkill radiucal
pkill harness

COMPARE="accounting proxy"
rm -f bin/accounting.log
rm -f bin/proxy.log

_getaux() {
    local upper
    upper=$(echo $1 | tr '[:lower:]' '[:upper:]')
    for f in $(ls $LOGS | sort); do
        cat ${LOGS}$f | grep "\[$upper\]" | cut -d " " -f 4-
    done
}

_getaux "proxy" | grep -v "^  Id" > bin/proxy.log
for o in accounting; do
    _getaux $o | \
        sed "s/^  //g" | cut -d " " -f 1,3 | \
        sed "s/^Access/ Access/g" | \
        sed "s/^UDPAddr/ UDPAddr/g" | \
        sed "s/^Id/ Id/g" | \
        cut -d " " -f 1,2 | \
        sort >> bin/$o.log
done

for d in $(echo $COMPARE); do
    diff --color -w -u expected/$CONF.$d.log bin/$d.log
    if [ $? -ne 0 ]; then
        echo "integration test failed ($d $1)"
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
_checks "client failed auth check" 2
echo "stdout checks passed"
sleep 3
echo "$CONF is completed..."
