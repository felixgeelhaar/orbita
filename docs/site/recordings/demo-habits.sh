#!/bin/bash
# Habits Demo Recording Script
ORBITA="./orbita-demo"

type_cmd() {
    echo -n "$ "
    for ((i=0; i<${#1}; i++)); do
        echo -n "${1:$i:1}"
        sleep 0.05
    done
    echo
    sleep 0.3
}

clear
sleep 0.5

# Show habit commands
type_cmd "orbita habit --help"
$ORBITA habit --help
sleep 1.5

# List habits
type_cmd "orbita habit list"
$ORBITA habit list
sleep 1.5

# Show create help
type_cmd "orbita habit create --help"
$ORBITA habit create --help
sleep 2

echo
echo "✨ Build consistent habits with streak tracking!"
sleep 2
