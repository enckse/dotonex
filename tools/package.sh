#!/bin/bash
FILES="netconf.py configure.sh reports.sh"
_name() {
    echo "$1" | cut -d "." -f 1 | sed "s#/##g;s#[_]##g"
}
_gen() {
    filevar="files"
    echo "// this file is auto-generated, do NOT edit it"
    echo "package main"
    echo
    echo "var ("
    for f in $FILES; do
        fname=$(_name "$f")
        echo "    // $f"
        echo "    $fname = []string{}"
    done
    echo "    // all files"
    echo "    $filevar = []*embedded{}"
    echo ")"
    echo
    echo "func init() {"
    for f in $FILES; do
        fname=$(_name "$f")
        echo "    // $fname script"
        cat $f | sed "s/^/    $fname = append($fname, \`/g;s/$/\`)/g"
        bname=$(basename $f | sed "s/\.sh$//g")
        name="${fname}Script"
        echo "    // $fname embedded object"
        echo "    $filevar = append(files, &embedded{content: $fname, name: \"$bname\"})"
    done
    echo "}"
}
_gen | sed "s/^    /\t/g"
