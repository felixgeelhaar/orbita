#!/bin/bash
# Meetings Demo Recording Script
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

# Show meeting commands
type_cmd "orbita meeting --help"
$ORBITA meeting --help
sleep 1.5

# List meetings
type_cmd "orbita meeting list"
$ORBITA meeting list
sleep 1.5

echo
echo "✨ Smart 1:1 scheduling from your terminal!"
sleep 2
