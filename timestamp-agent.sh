#!/bin/bash
# Test agent that logs timestamps to verify sequential execution

# Extract ticket number from the prompt
TICKET=$(echo "$1" | grep -o "ticket [0-9]" | grep -o "[0-9]")

# Get current timestamp
TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")

# Log to a file
echo "[$TIMESTAMP] Agent started for ticket $TICKET" >> agent-execution.log

# Simulate some work
sleep 1

# Add the agent's contribution to the output file
echo "echo \"[$TIMESTAMP] Agent $TICKET was here\"" >> example-output.sh

echo "Agent completed work on ticket $TICKET at $TIMESTAMP"

# Log completion
echo "[$TIMESTAMP] Agent completed for ticket $TICKET" >> agent-execution.log