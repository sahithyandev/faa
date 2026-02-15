# List all available commands (default)
default:
    @just --list

# Build the faa CLI binary
build:
    go build -o bin/faa ./cmd/localhost-dev

# Run tests
test:
    go test -v ./...

# Clean build artifacts
clean:
    rm -rf bin/

# Install the binary to GOPATH/bin
install:
    go install ./cmd/localhost-dev

# Run all checks (test and build)
all: test build
