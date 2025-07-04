# Contributing to Project Manager TUI

Thank you for your interest in contributing to Project Manager TUI! We welcome contributions from the community.

## How to Contribute

### Reporting Issues

- Check if the issue has already been reported
- Use the issue templates when available
- Include steps to reproduce the issue
- Provide your environment details (OS, Go version)

### Pull Requests

1. Fork the repository
2. Create a new branch for your feature/fix
3. Make your changes
4. Run tests and linting: `make test && make lint`
5. Commit your changes with clear commit messages
6. Push to your fork and submit a pull request

### Development Setup

1. Install Go 1.21 or later
2. Clone the repository
3. Install dependencies: `go mod download`
4. Run the application: `go run .`

### Code Style

- Follow standard Go formatting (use `gofmt`)
- Run `golangci-lint` before submitting
- Write clear, self-documenting code
- Add comments for exported functions
- Keep functions focused and small

### Testing

- Write tests for new features
- Ensure existing tests pass
- Test edge cases
- Run: `go test ./...`

### Commit Messages

- Use clear and descriptive commit messages
- Start with a verb (Add, Fix, Update, etc.)
- Keep the first line under 50 characters
- Add detailed description if needed

### Questions?

Feel free to open an issue for any questions about contributing.