#!/bin/bash
# Tasks Demo - Shows task management workflow
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

# Show current tasks
type_cmd "orbita task list"
$ORBITA task list
sleep 1.2

# Create a high priority task
type_cmd "orbita task create \"Deploy hotfix to production\" -p high -d 30"
$ORBITA task create "Deploy hotfix to production" -p high -d 30
sleep 1

# Quick add with natural language
type_cmd "orbita add \"Review PR from Alice tomorrow morning\""
$ORBITA add "Review PR from Alice tomorrow morning"
sleep 1

# Show updated list
type_cmd "orbita task list"
$ORBITA task list
sleep 1.2

# Complete a task
type_cmd "orbita done \"Deploy hotfix\""
$ORBITA done "Deploy hotfix"
sleep 0.8

echo
echo "Tasks managed efficiently from your terminal!"
sleep 1.5
