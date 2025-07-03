# Project Manager TUI

A terminal user interface for managing sequential coding agent execution using Bubble Tea.

## Features

- Automatic detection of specification files
- Interactive file picker for missing files
- Agent selection (claude or custom command)
- Sequential ticket execution with configurable delays
- Real-time progress tracking with output window
- Exponential backoff for API errors
- Visual countdown between agent executions

## Installation

1. Install Go (1.21 or later)
2. Install dependencies:
```bash
go mod download
```

## Usage

Run the application:
```bash
go run .
```

The application will:
1. Check for required files in the `specifications/` folder
2. If files are missing, let you select them using a file picker
3. Ask you to choose the coding agent
4. Show a confirmation screen
5. Execute agents sequentially for each ticket

## File Structure

- `specifications/specification.md` - Project specification
- `specifications/tickets.md` - Individual tickets for agents
- `specifications/standard-prompt.md` - Base prompt for all agents

## Controls

- `↑/↓` or `j/k` - Navigate options
- `Enter` - Select/Confirm
- `Tab` - Focus text input (when selecting custom agent)
- `q` or `Ctrl+C` - Quit

## Testing Without Agents

For testing without relying on AI agents, you can use a simple shell script as the "agent". When prompted to select an agent, choose "Other" and enter one of these commands:

### Option 1: Simple echo command
```bash
bash -c 'echo "echo \"Agent worked on ticket\"" >> example-output.sh'
```

### Option 2: Create a test script
First create a mock agent script:
```bash
cat > mock-agent.sh << 'EOF'
#!/bin/bash
# Extract ticket number from the prompt
TICKET=$(echo "$1" | grep -o "ticket [0-9]" | grep -o "[0-9]")
echo "echo \"Agent $TICKET was here\"" >> example-output.sh
EOF
chmod +x mock-agent.sh
```

Then use `./mock-agent.sh` as your custom agent command.

### Option 3: Use a simple sleep command for testing flow
```bash
bash -c 'sleep 1 && echo "Task completed"'
```

This allows you to test the full flow of the project manager without depending on external AI services.

## Sequential Execution & Delays

The project manager ensures agents run sequentially with a configurable delay between executions:

- Default delay: 2 seconds between agents
- Exponential backoff: Delay doubles on API errors (max 30 seconds)
- Visual countdown shows remaining wait time
- Prevents API rate limiting issues

### Testing Sequential Execution

Use the included timestamp agent to verify sequential execution:

```bash
./timestamp-agent.sh
```

This script:
- Logs timestamps to `agent-execution.log`
- Shows exact start/end times for each agent
- Helps verify agents aren't running concurrently

Check the log after running:
```bash
cat agent-execution.log
```

You should see timestamps at least 2 seconds apart between agent completions and the next agent starting.

### Testing

The project includes a comprehensive test suite in the `test-scripts/` directory:

- **Mock Agents**: Simulate different agent behaviors (success, failure, debug)
- **Test Scripts**: Automated tests for various features
- **Kill File Testing**: Verify the agent termination mechanism

Available test agents:
1. **Debug Agent** (`test-scripts/debug-agent.sh`): Shows exactly what arguments are received
2. **Stdin Test** (`test-scripts/stdin-test.sh`): Tests reading prompt from stdin
3. **Failing Agent** (`test-scripts/failing-agent.sh`): Simulates API errors
4. **Mock Agent** (`test-scripts/mock-agent.sh`): Full-featured mock agent with kill file support

Run tests with:
```bash
./test-scripts/test-kill-mechanism.sh
./test-scripts/test-party-flow.sh
```

### Testing Error Handling

Use the failing agent to test error handling and exponential backoff:

```bash
./project-manager
# Select "Other" and enter: ./test-scripts/failing-agent.sh
```

This script simulates API overload errors. You should see:
- Failed tickets marked with ❌
- Delay increases after each failure (2s → 4s → 8s → 16s → 30s max)
- All tickets are attempted despite failures
- Visual countdown between retries