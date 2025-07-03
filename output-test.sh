#!/bin/bash
# Test agent that produces visible output

echo "=== OUTPUT TEST AGENT ==="
echo "This is line 1 of output"
echo "This is line 2 of output"
echo ""
echo "Reading prompt from stdin..."

# Read stdin
PROMPT=$(cat)

# Extract ticket number
TICKET=$(echo "$PROMPT" | grep -o "ticket [0-9]" | grep -o "[0-9]")

echo "Working on ticket: $TICKET"
echo "Prompt length: ${#PROMPT} characters"
echo ""
echo "Doing some work..."
sleep 1
echo "Work completed!"
echo ""
echo "Final status: SUCCESS"

# Also write to the output file
echo "echo \"Agent processed ticket $TICKET\"" >> example-output.sh

exit 0