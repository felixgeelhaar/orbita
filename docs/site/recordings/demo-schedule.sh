#!/bin/bash
# Schedule Demo - Shows schedule and planning workflow
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

# Show today's dashboard
type_cmd "orbita today"
$ORBITA today
sleep 1.5

# Show today's schedule
type_cmd "orbita schedule today"
$ORBITA schedule today
sleep 1.2

# Plan the day
type_cmd "orbita plan"
$ORBITA plan
sleep 1.2

# Show week view
type_cmd "orbita schedule week"
$ORBITA schedule week
sleep 1.5

echo
echo "Your schedule, organized from the terminal!"
sleep 1.5
