#!/bin/bash
for p in $(pidof hostapd); do
    kill -HUP $p
done
for p in $(pidof radiucal); do
    kill -2 $p
done
