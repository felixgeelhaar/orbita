#!/bin/bash
# Calendar Demo - Shows calendar connection and sync
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

# Show connected calendars
type_cmd "orbita auth list"
$ORBITA auth list
sleep 1.2

# Show sync status
type_cmd "orbita sync status"
$ORBITA sync status
sleep 1

# Export schedule
type_cmd "orbita export ical --today"
$ORBITA export ical --today
sleep 1

# Show calendar conflicts
type_cmd "orbita schedule conflicts"
$ORBITA schedule conflicts
sleep 1.2

echo
echo "Seamless calendar sync from the command line!"
sleep 1.5
