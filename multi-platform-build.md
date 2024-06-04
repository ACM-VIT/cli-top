# Multi-platform build instructions:
- Change the `GOOS` and `GOARCH` environment variables while running the `go build` command.

## Code:

```zsh
# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o cli-top-windows-amd64.exe main.go 

# MacOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o cli-top-macos-arm64 main.go 

# MacOS AMD64 (Apple Intel)
GOOS=darwin GOARCH=amd64 go build -o cli-top-macos-amd64 main.go 

# Linux ARM64 (Android)
GOOS=linux GOARCH=arm64 go build -o cli-top-linux-arm64 main.go 

# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o cli-top-linux-amd64 main.go 

```
