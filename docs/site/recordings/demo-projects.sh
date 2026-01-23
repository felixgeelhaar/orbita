#!/bin/bash
# Projects Demo - Shows project management workflow
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

# List projects
type_cmd "orbita project list"
$ORBITA project list
sleep 1.2

# Create a new project
type_cmd "orbita project create \"Mobile App v2\" --due 2026-04-15"
$ORBITA project create "Mobile App v2" --due 2026-04-15
sleep 1

# Show project details
type_cmd "orbita project list"
$ORBITA project list
sleep 1.2

# Show project insights
type_cmd "orbita insights projects"
$ORBITA insights projects
sleep 1.2

echo
echo "Organize work into projects with milestones!"
sleep 1.5
