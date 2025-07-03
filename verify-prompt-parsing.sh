#!/bin/bash
# Verify that the mock agent can parse the prompt with kill file instruction

echo "Testing prompt parsing..."

PROMPT="You are a party planning agent! 🎉 Your job is to help create an amazing virtual party script.

Please follow these guidelines:
1. Read the specification.md carefully to understand the party theme
2. Work ONLY on your assigned ticket from tickets.md
3. Each agent has a unique role in making the party awesome
4. Keep your contributions fun, colorful, and quick to execute
5. Make sure to append to the existing party.sh file (don't overwrite!)
6. The party should get progressively more exciting with each agent's contribution

Remember: This is a celebration! Have fun with it! 🎊 Please use the documentation in the specifications folder, especially the specification.md and the tickets.md. Please work on ticket 2. As your final task, create a file named 'killmenow.md' containing either 'success' or 'failure' to indicate whether you successfully completed the task."

echo "Extracting ticket number from prompt..."
TICKET=$(echo "$PROMPT" | grep -o "ticket [0-9]" | grep -o "[0-9]")
echo "Extracted ticket number: $TICKET"

if [ "$TICKET" = "2" ]; then
    echo "✅ Successfully extracted ticket number from complex prompt!"
else
    echo "❌ Failed to extract ticket number"
fi