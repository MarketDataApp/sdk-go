#!/bin/bash

# Define paths relative to the script location
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
PROJECT_ROOT="${SCRIPT_DIR}/.."
COVERAGE_PROFILE="${SCRIPT_DIR}/coverage.out"
COVERAGE_HTML="${SCRIPT_DIR}/coverage.html"

# Ensure we're in the project root directory
cd "${PROJECT_ROOT}"

# Run tests for all packages including the root, and collect coverage across all packages
go test -timeout 100s -covermode=atomic -coverpkg=./... -coverprofile="${COVERAGE_PROFILE}" ./... || exit 1

# Generate a human-readable HTML coverage report
go tool cover -html="${COVERAGE_PROFILE}" -o "${COVERAGE_HTML}"

echo "Coverage report generated at ${COVERAGE_HTML}"