#!/bin/bash
# Setup Demo Data for Recordings
# This creates realistic demo data for the marketing site recordings

ORBITA="./orbita-demo"

echo "Setting up demo data..."

# Clear any existing demo data (if the command exists)
# $ORBITA task clear --force 2>/dev/null || true

# Create sample tasks with various priorities and durations
$ORBITA task create "Review Q1 roadmap with team" -p high -d 60 2>/dev/null
$ORBITA task create "Fix authentication bug in login flow" -p high -d 45 2>/dev/null
$ORBITA task create "Write API documentation for v2 endpoints" -p medium -d 90 2>/dev/null
$ORBITA task create "Refactor database queries for performance" -p medium -d 120 2>/dev/null
$ORBITA task create "Update dependencies to latest versions" -p low -d 30 2>/dev/null
$ORBITA task create "Code review: PR #847 - new search feature" -p high -d 30 2>/dev/null

# Create habits
$ORBITA habit create "Morning standup notes" --frequency daily 2>/dev/null
$ORBITA habit create "Weekly code review" --frequency weekly 2>/dev/null
$ORBITA habit create "Exercise" --frequency daily 2>/dev/null
$ORBITA habit create "Read technical articles" --frequency daily 2>/dev/null

# Create meetings
$ORBITA meeting create "Alice Chen" --cadence weekly 2>/dev/null
$ORBITA meeting create "Bob Smith" --cadence biweekly 2>/dev/null
$ORBITA meeting create "Carol Davis" --cadence monthly 2>/dev/null

# Create projects
$ORBITA project create "Q1 Product Launch" --due 2026-03-31 2>/dev/null
$ORBITA project create "API v2 Migration" --due 2026-02-28 2>/dev/null

echo "Demo data setup complete!"
