#!/bin/bash
FILES="netconf.py configure.sh reports.sh users/__config__.py users/__init__.py"
_name() {
    echo "$1" | cut -d "." -f 1 | sed "s#/##g;s#[_]##g"
}
_gen() {
    echo "package main"
    echo
    echo "const ("
    for f in $FILES; do
        fname=$(_name "$f")
        echo "    $fname = \`"
        cat $f
        echo "\`"
    done
    echo ")"
    echo
    allvars=""
    echo "var ("
    for f in $FILES; do
        fname=$(_name "$f")
        bname=$(basename $f | sed "s/\.sh$//g")
        exc="false"
        if echo "$f" | grep -q "\.sh"; then
            exc="true"
        fi
        dst=""
        d=$(echo "$f" | cut -d "/" -f 1)
        if [[ "$d" != "$f" ]]; then
            dst="$d/"
        fi
        srv="false"
        if echo "$f" | grep -q "reports\.sh"; then
            srv="true"
        fi
        name="${fname}Script"
        echo "    ${name} = &embedded{content: $fname, name: \"$bname\", exec: $exc, dest: \"$dst\", server:$srv}"
        allvars="$name $allvars"
    done
    echo "    files = []*embedded{"
    for v in $allvars; do
        echo "        $v,"
    done
    echo "    }"
    echo ")"
}
_gen
