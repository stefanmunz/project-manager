#!/bin/bash
# Test the folder selection feature

echo "Testing Folder Selection Feature"
echo "================================"

# Save current directory
ORIGINAL_DIR=$PWD

# Change to parent directory if we're in test-scripts
if [[ $PWD == */test-scripts ]]; then
    cd ..
fi

# Create a test folder structure
TEST_DIR="test-input-folder"
rm -rf $TEST_DIR
mkdir -p $TEST_DIR

# Copy test files to the new folder
cp input/specification.md $TEST_DIR/
cp input/tickets.md $TEST_DIR/
cp input/standard-prompt.md $TEST_DIR/

echo ""
echo "Created test folder: $TEST_DIR"
echo "Files in test folder:"
ls -la $TEST_DIR/

echo ""
echo "NOTE: When the project manager starts:"
echo "1. You'll see two options: 'Use default input folder' and 'Other'"
echo "2. Select 'Other' using arrow keys"
echo "3. Enter the folder path: $TEST_DIR"
echo "4. The tool should find all files in that custom location"
echo ""
echo "Press Enter to start the project manager..."
read

# Run project manager
./project-manager

# Cleanup
echo ""
echo "Cleaning up test folder..."
rm -rf $TEST_DIR

cd $ORIGINAL_DIR