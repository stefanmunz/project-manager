#!/bin/bash
# Simple test that just echoes success

echo "Test agent executed successfully!"
echo "Received prompt of length: ${#1}"
echo "First 100 chars of prompt: ${1:0:100}..."
exit 0