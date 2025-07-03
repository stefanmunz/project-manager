# Kill File Mechanism

The project manager now uses a "kill file" mechanism to handle agents that don't automatically terminate after completing their tasks (like Claude).

## How It Works

1. **Modified Prompt**: The agent prompt now includes instructions to create a `killmenow.md` file as the final task
2. **Async Execution**: Agents are started asynchronously using `cmd.Start()` instead of waiting synchronously
3. **File Monitoring**: The project manager monitors for the `killmenow.md` file every 500ms
4. **Process Termination**: When the file is detected:
   - Its content is read (should contain "success" or "failure")
   - The agent process is terminated via `Process.Kill()`
   - The kill file is deleted
   - The ticket is marked as completed or failed based on the file content

## Testing

Run the test scripts to verify the mechanism:
- `./test-kill-mechanism.sh` - Tests the basic kill file detection
- `./test-full-flow.sh` - Tests the complete project manager flow
- `./verify-prompt-parsing.sh` - Verifies prompt parsing with kill file instructions

## Example Kill File Content

```
success
```

or

```
failure
```

The content is case-insensitive - any file containing "success" will mark the ticket as completed.