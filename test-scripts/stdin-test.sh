#!/bin/bash
# Test agent that reads from stdin
# NOTE: This script expects input via stdin - run it through project-manager!
# Running directly will wait forever. Try: echo "test" | ./stdin-test.sh

echo "=== STDIN TEST AGENT ==="
echo "Reading from stdin..."
echo ""

# Read all of stdin
PROMPT=$(cat)

echo "Received prompt via stdin:"
echo "Length: ${#PROMPT} characters"
echo "---"
echo "$PROMPT" | head -5
echo "..."
echo "---"

# Extract ticket number
TICKET=$(echo "$PROMPT" | grep -o "ticket [0-9]" | grep -o "[0-9]")
echo "Detected ticket number: $TICKET"

# Success
echo "Test completed successfully!"
exit 0