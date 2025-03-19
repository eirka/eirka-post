# Eirka-Post Go Application Guidelines

## Build & Test Commands
```bash
# Build and run application
go build
./eirka-post  # or go run main.go

# Run all tests
go test ./...

# Run specific test
go test -v ./path/to/package -run TestName

# Test with coverage
go test -cover ./...
```

## Code Style Guidelines
- **Imports**: Standard lib first, third-party next, local last (separated by blank lines)
- **Naming**: PascalCase for exports, camelCase for private, Models: `ModelNameModel`, Controllers: `ActionController`
- **Error Handling**: Use custom errors, check with `if err != nil`, controllers set metadata with `c.Error(err).SetMeta()`
- **Testing**: Table-driven tests, descriptive names, mock external dependencies
- **Structure**: Clean separation between controllers (HTTP), models (business logic), and utils (helpers)
- **Gin Framework**: Follow standard Gin conventions for context handling and middleware
- **Database**: Explicit transaction handling with Begin/Commit/Rollback pattern

Remember to maintain the existing error handling approach using the custom error types from the eirka-libs package.
