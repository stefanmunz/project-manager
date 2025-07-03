#!/bin/bash
# Test script to verify claude command format

echo "Testing Claude command format..."
echo "================================"

# Simulate the command that would be executed
PROMPT="This is the standard prompt. Please use the documentation in the specifications folder, especially the specification.md and the tickets.md. Please work on ticket 1"

echo "Command that would be executed:"
echo 'claude --dangerously-skip-permissions "'"$PROMPT"'"'
echo ""

echo "Testing with a simple echo command:"
echo "$PROMPT" | xargs -0 echo "Received prompt:"