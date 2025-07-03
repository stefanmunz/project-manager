#!/bin/bash
# Mock agent for testing the project manager - Party Edition! 🎉
# Extracts ticket number from the prompt and adds party elements to party.sh

# Extract ticket number from the prompt
TICKET=$(echo "$1" | grep -o "ticket [0-9]" | grep -o "[0-9]")

# Simulate some work
sleep 1

# Create party contributions based on ticket number
case $TICKET in
    1)
        # First agent starts the party
        echo '#!/bin/bash' > party.sh
        echo 'echo -e "\033[1;35m🎉 Agent 1 arrives with balloons! 🎈\033[0m"' >> party.sh
        echo 'sleep 1' >> party.sh
        chmod +x party.sh
        echo "🎈 Agent 1: Party started! Created party.sh"
        ;;
    2)
        # Second agent brings the music
        echo 'echo -e "\033[1;36m🎵 Agent 2 starts the music! 🎶\033[0m"' >> party.sh
        echo 'echo "  ♪ ♫ ♪ ♫ ♪ ♫"' >> party.sh
        echo 'sleep 1' >> party.sh
        echo "🎵 Agent 2: Music is playing!"
        ;;
    3)
        # Third agent brings fireworks for the finale
        echo 'echo -e "\033[1;33m🎆 Agent 3 brings fireworks! 🎇\033[0m"' >> party.sh
        echo 'echo -e "\033[1;32m🎊 ALL AGENTS: Party complete! What a celebration! 🎊\033[0m"' >> party.sh
        echo "🎆 Agent 3: Fireworks launched! Party complete!"
        ;;
    *)
        echo "Unknown ticket number: $TICKET"
        ;;
esac

# Create kill file to signal completion
echo "success" > killmenow.md
echo "Created kill file to signal completion"