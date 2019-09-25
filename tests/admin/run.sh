#!/bin/bash
FILES=bin/
CONFIG=config/
PASS=testtesttesttesttesttesttesttest
PASSFILE=${CONFIG}passwords
cp ../../target/release/radiucal-admin .
cp ../../radiucal-lua-bridge .

PATH=$PATH:$PWD
export PATH
_radiucal-admin() {
    echo $1 | ./radiucal-admin ${@:2} --pass=$PASS
}

rm -rf $FILES
rm -rf $CONFIG
mkdir -p $FILES
mkdir -p $CONFIG

cp passwords $CONFIG
cp *.lua $CONFIG

_radiucal-admin "" netconf

failure=0
for f in audit.csv manifest eap_users sysinfo.csv segment-diagram.dot segments.csv; do
    echo "checking $f"
    diff -u $f ${FILES}$f
    if [ $? -ne 0 ]; then
        echo "FAILED"
        failure=1
    fi
done

echo "test,abc
test2,xyz
test3,111
test4,999" > ${PASSFILE}
_radiucal-admin "" enc
rm $PASSFILE
t=$(_radiucal-admin "test" passwd | grep md4 | cut -d " " -f 3)
rm $PASSFILE
t2=$(_radiucal-admin "test2" passwd | grep md4 | cut -d " " -f 3)
rm $PASSFILE
zzzz=$(_radiucal-admin "zzzz" useradd | grep md4 | cut -d " " -f 3)
rm $PASSFILE
_radiucal-admin "" dec

TEST_PASS=${FILES}pwd.
echo "test2,$t2
test3,111
test4,999
test,$t
zzzz,$zzzz" | sort > ${TEST_PASS}exp
cat $PASSFILE | sort > ${TEST_PASS}act
diff -u ${TEST_PASS}exp ${TEST_PASS}act
if [ $? -ne 0 ]; then
    echo "invalid password changes"
    failure=1
fi

exit $failure
