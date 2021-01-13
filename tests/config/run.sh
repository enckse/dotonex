#!/bin/bash

EXPECT=expected/
REPO=$PWD/repo/
BIN=bin/
mkdir -p $BIN
RESULTS=${BIN}log
CHECK=$RESULTS.check
echo > $RESULTS
KEY=${REPO}server.local
TOKEN=${REPO}user.name/token.local
EAP=${REPO}eap_users
KNOWN=${REPO}known.local
rm -f $KEY $TOKEN $EAP $KNOWN

_command() {
    python ../../tools/dotonex-config $1 $REPO ${@:2} echo '{{"username":"user.name"}}' >> $RESULTS
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

_diff_known() {
    sed -i "s/$(date +%Y-%m-%d)/DATE/g" ${KNOWN}
    _diff $1 $KNOWN
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
_diff_known known.abcdef
_command validate --token token --mac aabbccddeeff
_diff_known known.token
_command validate --token abcdef --mac 1122334455aa
_diff_known known.token
_diff_eap user

cat $RESULTS | grep -v "$PWD" > $CHECK
diff -u $CHECK ${EXPECT}log
if [ $? -ne 0 ]; then
    echo "incorrect execution"
    exit 1
fi
