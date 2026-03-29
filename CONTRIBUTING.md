# Contributing

Thank you for your interest in contributing to the QNAP File Station API SDK for Go!

## Development Setup

1. Clone the repository
2. Ensure you have Go 1.26+ installed
3. Run tests: `go test ./...`
4. Run linter: `golangci-lint run`

## Adding New Features

1. Fork the repository
2. Create a feature branch
3. Write tests for your feature
4. Implement the feature
5. Ensure all tests pass
6. Submit a pull request

## Code Style

- Follow Go conventions and idioms
- Use `gofmt` for formatting
- Add godoc comments for exported types and functions
- Keep functions small and focused
- Write tests for all new functionality

## Testing

We use table-driven tests and mock servers for testing. See `api/client_test.go` and `filestation/file_test.go` for examples.
