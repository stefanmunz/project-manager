#!/bin/bash
# Test the kill file mechanism

echo "🔧 Testing Kill File Mechanism"
echo "=============================="

# Change to parent directory if we're in test-scripts
if [[ $PWD == */test-scripts ]]; then
    cd ..
fi

# Clean up any existing files
rm -f *-party.sh killmenow.md

# Get current day and time for filename
DAY=$(date +%A | tr '[:upper:]' '[:lower:]')
TIME=$(date +%H:%M)
FILENAME="${DAY}-${TIME}-party.sh"

# Test a single agent execution
echo "Testing single agent with kill file..."
echo "Expected filename: $FILENAME"

# Read the standard prompt
PROMPT_CONTENT=$(cat input/standard-prompt.md)

# Create the full prompt with kill file instruction
FULL_PROMPT="$PROMPT_CONTENT Please use the documentation in the input folder, especially the specification.md and the tickets.md. Please work on ticket 1. As your final task, create a file named 'killmenow.md' containing either 'success' or 'failure' to indicate whether you successfully completed the task."

echo "Executing mock agent..."
./test-scripts/mock-agent.sh "$FULL_PROMPT" &
AGENT_PID=$!

echo "Agent started with PID: $AGENT_PID"

# Monitor for kill file
echo "Monitoring for kill file..."
while [ ! -f killmenow.md ]; do
    sleep 0.5
    echo -n "."
done

echo ""
echo "Kill file detected!"
CONTENT=$(cat killmenow.md)
echo "Content: $CONTENT"

# Kill the agent (if still running)
if ps -p $AGENT_PID > /dev/null; then
    echo "Killing agent process..."
    kill $AGENT_PID
else
    echo "Agent already terminated"
fi

# Clean up
rm -f killmenow.md

echo ""
echo "✅ Test completed successfully!"

# Check if party script was created with timestamp
if [ -f "$FILENAME" ]; then
    echo ""
    echo "Party script created: $FILENAME"
    echo "Contents:"
    cat "$FILENAME"
else
    echo ""
    echo "❌ Error: $FILENAME was not created!"
fi