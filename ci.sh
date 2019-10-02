#!/bin/bash
set -e
set -x

# Check versions and current directory
node -v
npm -v
go version
golangci-lint --version

# Print the current directory contents
ls -la

# Run linter
golangci-lint run

# Run tests
go test -race -v -bench=. -coverprofile=coverage.txt -covermode=atomic ./...

# Make
make clean
make build/static/index.html
make

if [[ -z "$(git status --porcelain)" ]]; then
    # Working directory clean
    echo "Git status is clean"
else
    echo "Git status is not clean and contains uncommited changes"
    echo "Please make sure there are no changes"
    exit 1
fi
