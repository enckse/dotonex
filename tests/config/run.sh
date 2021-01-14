#!/bin/bash

EXPECT=expected/
REPO=$PWD/repo/
BIN=bin/
REPOBIN=${REPO}${BIN}
mkdir -p $BIN
RESULTS=${BIN}log
echo > $RESULTS
EAP=${REPOBIN}eap_users
HASH1=${REPOBIN}b5fe2db507cc5ac540493d48fbd5fe33.db
HASH2=${REPOBIN}3607e48be4f77269241d049a8765cb18.db
HASH1EXP=hash1
HASH2EXP=hash2
rm -f ${REPOBIN}*

_command() {
    ../../dotonex-compose --mode $1 --repository $REPO ${@:2} echo '{"username":"user.name"}' >> $RESULTS 2>&1
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

_diff_hash() {
    local hash
    test -e $1
    _diff $2 $1
    hash=$(cat ${REPOBIN}user.name.db)
    if [[ "$hash" != "$3" ]]; then
        echo "hash mismatch $hash != $3"
        exit 1
    fi
}

_diff_token_hash() {
    _diff_hash $HASH1 $HASH1EXP token
    _diff_hash $HASH2 $HASH2EXP token
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

_command validate --token abcdef --mac 1122334455aa
_diff_hash $HASH1 $HASH1EXP abcdef
_command validate --token token --mac aabbccddeeff
_diff_token_hash
_command validate --token abcdef --mac 1122334455aa
_diff_token_hash
_diff_eap user

diff -u $RESULTS ${EXPECT}log
if [ $? -ne 0 ]; then
    echo "incorrect execution"
    exit 1
fi
