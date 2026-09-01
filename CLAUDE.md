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

## Security Guidelines

### File Upload Security
- **Filename Validation**: Always validate file extensions and reject filenames with suspicious patterns
- **MIME Type Verification**: Check that the file's MIME type matches its extension
- **File Signature Validation**: Validate file signature headers (magic bytes) of uploaded files
- **Content Verification**: For images, validate dimensions, format, and file integrity
- **Size Constraints**: Enforce strict size limits for uploads
- **Extension Whitelist**: Only allow specific file extensions
- **Avoid Multiple Extensions**: Reject files with multiple extensions to prevent extension-based attacks
- **Suspect Files**: Check for suspiciously small files or incomplete file formats

### Testing Security
- Write tests for security validation failures
- Create test cases for expected bypasses
- Use table-driven tests for multiple security test cases

## Test prerequisites
- No /etc/pram/pram.conf -> config/config.go defaults to /tmp/eirka/{src,thumb,avatars}/; utils/testdirs_test.go init() creates them.
- utils init() (image.go, webm.go) panics without convert, ffmpeg, ffprobe on PATH; controllers imports utils so needs them.
- middleware/sfs_test.go asserts on live api.stopforumspam.org responses (no skip guard) - fails after a 10s timeout offline.
- No live DB/redis: go-sqlmock and redigomock. No CI in repo; default branch master.
- Reset test storage: find /tmp/eirka -mindepth 1 -delete

## Fleet conventions
Config is JSON from /etc/pram/pram.conf; auth is eirka-libs/user JWT (user.Auth(bool) in main.go, cookie set in controllers/login.go); tests use testify + go-sqlmock + redigomock (tempredis only transitively in go.sum, unused here).
