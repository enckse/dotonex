#!/bin/bash

EXPECT=expected/
REPO=$PWD/repo/
BIN=bin/
mkdir -p $BIN
RESULTS=${BIN}log
echo > $RESULTS
KEY=${REPO}server.local
TOKEN=${REPO}user.name/token.local
EAP=${REPO}eap_users
rm -f $KEY $TOKEN $EAP

_command() {
    python ../../scripts/dotonex-config $1 $REPO ${@:2} >> $RESULTS
}

_diff() {
    diff -u ${EXPECT}$1 ${EAP}
    if [ $? -ne 0 ]; then
        echo "$1 failed"
        exit 1
    fi
}

# no password
_command build
if [ -e $EAP ]; then
    echo "should have failed, no password"
    exit 1
fi

_command server --hash "test"
_command server --hash "test"
_command server --hash "HASH"
_command build
_diff mabonly

echo "token" > $TOKEN
_command build
_diff user

