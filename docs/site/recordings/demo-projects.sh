#!/bin/bash
# Projects Demo Recording Script
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

# Show project commands
type_cmd "orbita project --help"
$ORBITA project --help
sleep 1.5

# List projects
type_cmd "orbita project list"
$ORBITA project list
sleep 1.5

# Show create help
type_cmd "orbita project create --help"
$ORBITA project create --help
sleep 2

echo
echo "✨ Organize work into projects with milestones!"
sleep 2
