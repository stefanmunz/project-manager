#!/bin/bash
# Test the party flow by directly calling mock agents

echo "🎊 Testing Party Manager Flow 🎊"
echo "==============================="

# Change to parent directory if we're in test-scripts
if [[ $PWD == */test-scripts ]]; then
    cd ..
fi

# Clean up
rm -f party.sh

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

if [ -f party.sh ]; then
    echo "Running party.sh..."
    echo ""
    ./party.sh
else
    echo "❌ Error: party.sh was not created!"
fi