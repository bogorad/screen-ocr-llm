# Show a chooser when no recipe is specified.
default:
    @just --choose

# Build the Windows GUI app binary.
build-windows:
    go build -ldflags "-H=windowsgui" -o screen-ocr-llm.exe ./src/main

# Build the Linux CLI-only binary.
build-linux:
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ocr-tool ./src/cmd/cli

# Alias for Linux CLI build.
build-cli-linux:
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ocr-tool ./src/cmd/cli

# Run all tests.
test:
    go test ./...
