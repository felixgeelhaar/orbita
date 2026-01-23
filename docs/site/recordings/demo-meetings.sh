#!/bin/bash
# Meetings Demo - Shows 1:1 meeting management
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

# List current meetings
type_cmd "orbita meeting list"
$ORBITA meeting list
sleep 1.2

# Create a new 1:1
type_cmd "orbita meeting create \"David Kim\" --cadence weekly"
$ORBITA meeting create "David Kim" --cadence weekly
sleep 1

# Show meeting details
type_cmd "orbita meeting list"
$ORBITA meeting list
sleep 1.2

# Adapt meeting frequency
type_cmd "orbita adapt meetings"
$ORBITA adapt meetings
sleep 1

echo
echo "Smart 1:1 scheduling from your terminal!"
sleep 1.5
