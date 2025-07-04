# Claude Development Guidelines

## Setup

After cloning the repository, install development tools:

```bash
make install-tools
```

This installs golangci-lint and other required development tools.

## Pre-Commit Checklist

Before committing any changes, always run the following commands:

1. **Run tests**

   ```bash
   go test -v ./...
   ```

2. **Run go vet** (catches suspicious constructs)

   ```bash
   go vet ./...
   ```

3. **Run golangci-lint** (comprehensive linting including errcheck)

   ```bash
   golangci-lint run
   ```

4. **Update CLAUDE.md** for meaningful changes
   Claude.md mis the onboarding document for future coding agents. Please create the information there as local as possible, keep the central CLAUDE.md only for central information

All commands must pass without errors before committing. The CI will run these same checks, so catching issues locally saves time.

Note: `go vet` catches some issues but doesn't include all linters like `errcheck`. The `golangci-lint` tool is more comprehensive and is what the CI uses.
