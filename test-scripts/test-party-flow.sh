#!/bin/bash
# Test the party flow by directly calling mock agents

echo "🎊 Testing Party Manager Flow 🎊"
echo "==============================="

# Change to parent directory if we're in test-scripts
if [[ $PWD == */test-scripts ]]; then
    cd ..
fi

# Clean up old party files
rm -f *-party.sh

# Get current day and time for filename
DAY=$(date +%A)
TIME=$(date +%H:%M)
FILENAME="${DAY}-${TIME}-party.sh"

echo "Creating party script: $FILENAME"

# Read the standard prompt
PROMPT_CONTENT=$(cat input/standard-prompt.md)

# Test each agent in sequence
for i in 1 2 3; do
    echo ""
    echo "📢 Launching Agent $i..."
    FULL_PROMPT="$PROMPT_CONTENT Please use the documentation in the input folder, especially the specification.md and the tickets.md. Please work on ticket $i"
    
    # Call mock agent with the prompt as an argument
    ./test-scripts/mock-agent.sh "$FULL_PROMPT"
    
    # Small delay between agents
    sleep 1
done

echo ""
echo "🎭 Party planning complete! Let's see the party:"
echo "================================================"

if [ -f "$FILENAME" ]; then
    echo "Running $FILENAME..."
    echo ""
    ./"$FILENAME"
else
    echo "❌ Error: $FILENAME was not created!"
fi