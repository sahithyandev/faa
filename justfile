# Build the faa CLI binary
build:
    go build -o bin/faa ./cmd/faa

# Run tests
test:
    go test -v ./...

# Clean build artifacts
clean:
    rm -rf bin/

# Install the binary to GOPATH/bin
install:
    go install ./cmd/faa

# Run all checks (test and build)
all: test build
