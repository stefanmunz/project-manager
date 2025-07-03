# Test Scripts

This directory contains test scripts for the Project Manager application.

## Mock Agents

These scripts simulate different agent behaviors:

- **`mock-agent.sh`** - Main mock agent that implements the party demo and kill file mechanism
- **`debug-agent.sh`** - Shows exactly what arguments are received (useful for debugging)
- **`failing-agent.sh`** - Always fails with "server overload" error (tests error handling)
- **`stdin-test.sh`** - Tests stdin input handling

## Test Scripts

These scripts test various aspects of the system:

- **`test-party-flow.sh`** - Tests the complete party demo flow end-to-end
- **`test-kill-mechanism.sh`** - Tests the kill file detection and process termination
- **`verify-prompt-parsing.sh`** - Verifies prompt parsing with complex inputs

## Usage

Most test scripts can be run directly:

```bash
./test-scripts/test-kill-mechanism.sh
```

For testing with the project manager, use mock-agent.sh as the custom command:

```
./test-scripts/mock-agent.sh
```