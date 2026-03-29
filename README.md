# QNAP File Station API SDK for Go

A comprehensive Go SDK for interacting with QNAP QTS File Station API v5.

## Features

- **Authentication**: SID-based session management with auto-renewal
- **File Operations**: Upload, download, delete, rename, copy, move files
- **Folder Operations**: Create, list, delete, rename, copy, move folders
- **Search**: Advanced file and folder search with filters
- **Share Links**: Create, list, and manage share links
- **Context Support**: Full context support for cancellation and timeouts
- **Type Safety**: Strongly typed structs and responses
- **Error Handling**: Comprehensive error handling with QNAP-specific error codes

## Requirements

- Go 1.26 or higher

## Installation

```bash
go get github.com/fatelei/qnap-filestation
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/fatelei/qnap-filestation/pkg/api"
    "github.com/fatelei/qnap-filestation/pkg/filestation"
)

func main() {
    // Create client
    client, err := api.NewClient(&api.Config{
        Host:     "192.168.1.100",
        Port:     8080,
        Username: "admin",
        Password: "password",
        Insecure: true, // For self-signed certificates
    })
    if err != nil {
        log.Fatal(err)
    }

    // Login
    ctx := context.Background()
    if err := client.Login(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Logout(ctx)

    // Create FileStation service
    fs := filestation.NewFileStationService(client)

    // List files
    files, err := fs.ListFiles(ctx, "/home", nil)
    if err != nil {
        log.Fatal(err)
    }

    for _, file := range files {
        fmt.Println(file.Name, file.Size)
    }
}
```

## Documentation

Full documentation is available at [docs/](docs/)

- [Authentication](docs/examples/authentication.md)
- [File Operations](docs/examples/file_operations.md)
- [Folder Operations](docs/examples/folder_operations.md)
- [Upload/Download](docs/examples/file_transfer.md)
- [Share Links](docs/examples/share_links.md)

## Development

### Requirements

- Go 1.21 or higher

### Running Tests

```bash
go test ./...
```

### Running with Coverage

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting

```bash
golangci-lint run
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Disclaimer

This SDK is not officially affiliated with or endorsed by QNAP Systems, Inc.
