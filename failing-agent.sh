#!/bin/bash
# Test agent that always fails with "server overload" error

# Extract ticket number from the prompt
TICKET=$(echo "$1" | grep -o "ticket [0-9]" | grep -o "[0-9]")

# Log the attempt
echo "[$(date +"%Y-%m-%d %H:%M:%S")] Attempting ticket $TICKET" >> agent-execution.log

# Simulate API overload error
echo "Repeated server overload with Opus model"
exit 1