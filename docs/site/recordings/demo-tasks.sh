#!/bin/bash
# Tasks Demo Recording Script
# Run: asciinema rec -c "./demo-tasks.sh" tasks-demo.cast

ORBITA="./orbita-demo"
DELAY=1

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

# Show task list
type_cmd "orbita task list"
$ORBITA task list
sleep $DELAY

# Create a new task
type_cmd "orbita task create \"Ship new feature\" -p high -d 60"
$ORBITA task create "Ship new feature" -p high -d 60
sleep $DELAY

# Show updated list
type_cmd "orbita task list"
$ORBITA task list
sleep $DELAY

# Complete a task
TASK_ID="559fa237"
type_cmd "orbita task complete $TASK_ID"
$ORBITA task complete $TASK_ID
sleep $DELAY

# Final list
type_cmd "orbita task list"
$ORBITA task list
sleep 2

echo
echo "✨ Tasks managed efficiently from the command line!"
sleep 2
