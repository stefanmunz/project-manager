#!/bin/bash
# Test the full project manager flow with kill file mechanism

echo "🎊 Testing Full Project Manager Flow with Kill File Mechanism 🎊"
echo "=============================================================="
echo ""
echo "This will test the project manager with mock agents that create kill files."
echo ""

# Clean up
rm -f party.sh killmenow.md

echo "Starting project manager..."
echo "Instructions:"
echo "1. It should find all specification files"
echo "2. Select 'Other (custom command)' and enter: ./mock-agent.sh"
echo "3. Press Enter to confirm and start"
echo "4. Watch as agents execute and are terminated via kill files"
echo ""
echo "Press Enter to continue..."
read

# Run the project manager
./project-manager