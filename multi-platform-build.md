# Multi-platform build instructions:
- Change the `GOOS` and `GOARCH` environment variables while running the `go build` command. To produce smaller binaries, include `-trimpath` and `-ldflags "-s -w"`.

## Code:

```zsh
# Windows AMD64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o cli-top-windows-amd64.exe main.go

# MacOS ARM64 (Apple Silicon)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o cli-top-macos-arm64 main.go

# MacOS AMD64 (Apple Intel)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o cli-top-macos-amd64 main.go

# Linux ARM64 (Android)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o cli-top-linux-arm64 main.go

# Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o cli-top-linux-amd64 main.go

```
