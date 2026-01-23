#!/bin/bash
# Calendar Demo Recording Script
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

# Show connected calendars
type_cmd "orbita auth list"
$ORBITA auth list
sleep 1.5

# Show auth connect help
type_cmd "orbita auth connect --help"
$ORBITA auth connect --help
sleep 2

echo
echo "✨ Connect Google, Microsoft, or Apple calendars!"
sleep 2
