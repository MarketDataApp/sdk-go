#!/bin/bash

# Define paths relative to the script location
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
PROJECT_ROOT="${SCRIPT_DIR}/.."
COVERAGE_PROFILE="${SCRIPT_DIR}/coverage.out"
COVERAGE_HTML="${SCRIPT_DIR}/coverage.html"

# Ensure we're in the project root directory
cd "${PROJECT_ROOT}"

# Initialize the coverage profile
echo "mode: atomic" > "${COVERAGE_PROFILE}"

# List all packages, excluding the tests directory itself
PACKAGES=$(go list ./... | grep -v "/tests")

# Run tests and collect coverage for each package
for PACKAGE in ${PACKAGES}; do
    go test -covermode=atomic -coverpkg=./... -coverprofile=profile.out ${PACKAGE}
    if [ -f profile.out ]; then
        cat profile.out | grep -v "mode: atomic" >> "${COVERAGE_PROFILE}"
        rm profile.out
    fi
done

# Run integration tests and include them in the coverage report
go test -covermode=atomic -coverpkg=./... -coverprofile=profile.out ./tests/...
if [ -f profile.out ]; then
    cat profile.out | grep -v "mode: atomic" >> "${COVERAGE_PROFILE}"
    rm profile.out
fi

# Generate a human-readable HTML coverage report
go tool cover -html="${COVERAGE_PROFILE}" -o "${COVERAGE_HTML}"

echo "Coverage report generated at ${COVERAGE_HTML}"
