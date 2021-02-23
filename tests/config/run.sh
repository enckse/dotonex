#!/bin/bash

EXPECT=expected/
REPO=$PWD/repo/
BIN=bin/
REPOBIN=${REPO}$BIN
mkdir -p $BIN
RESULTS=${BIN}log
echo > $RESULTS
EAP=${REPOBIN}eap_users
DB=${REPOBIN}dotonex.db
DBLOG=${BIN}db.log
rm -rf $REPOBIN $DBLOG
export DOTONEX_DEBUG="true"

_command() {
    ../../dotonex-compose --mode $1 --repository $REPO ${@:2} echo "{\"username\":\"$SET_USER\"}" >> $RESULTS 2>&1
}

_read() {
    go run db.go -database $DB | sort > $DBLOG
}

_diff() {
    diff -u ${EXPECT}$1 $2
    if [ $? -ne 0 ]; then
        echo "$1 != $2 failed"
        exit 1
    fi
}

_diff_eap() {
    _diff $1 ${EAP}
}

_diff_db() {
    _read > $DBLOG
    _diff $1 ${DBLOG}
}

# no password
_command rebuild
if [ -e $EAP ]; then
    echo "should have failed, no password"
    exit 1
fi

_command server --hash "test"
_command server --hash "test"
_command server --hash "HASH"
_command rebuild
_diff_eap mabonly
_diff_db serverhash
SET_USER=user.name
_command validate --token abcdef --mac aabbccddeeff
_diff_db token1
_command validate --token token --mac aabbccddeeff
_diff_db token2
_command validate --token abcdef --mac aabbccddeeff
_diff_db token2
_diff_eap user
_command validate --token abcdef --mac aabbccddee11
_diff_db token2
_diff_eap user
_command validate --token abcdef --mac aabbccddee1g
_diff_db token2
_diff_eap user

SET_USER=person.name
_command validate --token xxxxxx --mac aabbccdd1111
_diff_db token3
_diff_eap users
SET_USER=user.name
_command validate --token zzzzzz --mac aabbccddeeff
_diff_db token4
_diff_eap users2
_command validate --token zzzzzz --mac aabbccdd1111
_diff_db token4
_diff_eap users2

_command mac --mac aabbccddeeff
_command mac --mac aabbccdd1111
_command mac --mac 1234567890ab

diff -u $RESULTS ${EXPECT}log
if [ $? -ne 0 ]; then
    echo "incorrect execution"
    exit 1
fi
