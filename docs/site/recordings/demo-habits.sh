#!/bin/bash
# Habits Demo - Shows habit tracking workflow
ORBITA="./orbita-demo"

type_cmd() {
    echo -n "$ "
    for ((i=0; i<${#1}; i++)); do
        echo -n "${1:$i:1}"
        sleep 0.04
    done
    echo
    sleep 0.2
}

clear
sleep 0.3

# Show habit help
type_cmd "orbita habit --help"
$ORBITA habit --help
sleep 1.5

# Show habit create syntax
type_cmd "orbita habit create --help"
$ORBITA habit create --help
sleep 1.5

echo
echo "Build consistent habits with streak tracking!"
sleep 1.5
