#!/bin/bash
FILES="netconf.py configure.sh reports.sh users/__config__.py users/__init__.py"
_gen() {
    echo "package main"
    echo
    echo "const ("
    for f in $FILES; do
        fname=$(echo "$f" | cut -d "." -f 1 | sed "s#/##g")
        echo "    $fname = \`"
        cat $f
        echo "\`"
    done
    echo ")"
}
_gen
