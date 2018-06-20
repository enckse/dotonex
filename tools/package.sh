#!/bin/bash
FILES="netconf.py configure.sh reports.sh users/__config__.py users/__init__.py"
_name() {
    echo "$1" | cut -d "." -f 1 | sed "s#/##g;s#[_]##g"
}
_gen() {
    filevar="files"
    echo "package main"
    echo
    echo "var ("
    for f in $FILES; do
        fname=$(_name "$f")
        echo "    $fname = []string{}"
    done
    echo "    $filevar = []*embedded{}"
    echo ")"
    echo
    echo "func init() {"
    for f in $FILES; do
        fname=$(_name "$f")
        cat $f | sed "s/^/    $fname = append($fname, \`/g;s/$/\`)/g"
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
        echo "    $filevar = append(files, &embedded{content: $fname, name: \"$bname\", exec: $exc, dest: \"$dst\", server:$srv})"
    done
    echo "}"
}
_gen
