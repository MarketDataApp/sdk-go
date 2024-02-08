#!/bin/bash

# Create a temporary directory using mktemp
TMP_DIR=$(mktemp -d)

# Adjust to point to the project root from .scripts
SRC_DIR="$(dirname "$0")/.."
# Define the output directory for the markdown file within the .scripts directory
OUTPUT_DIR="$(dirname "$0")"
MAIN_FILE="indices_candles.go" # Main source file
TEST_FILE="indices_candles_test.go" # Corresponding test file

echo "Using temporary directory: $TMP_DIR"

# Copy the main file and its test file to the temporary directory
cp "$SRC_DIR/$MAIN_FILE" "$TMP_DIR"
cp "$SRC_DIR/$TEST_FILE" "$TMP_DIR"

# Optionally, copy any dependencies needed for documentation generation

# Run gomarkdoc on the temporary directory
gomarkdoc --output "$OUTPUT_DIR/sdk-go.md" "$TMP_DIR"

# Clean up: remove the temporary directory
# rm -rf "$TMP_DIR"

echo "Documentation generated for $MAIN_FILE and $TEST_FILE"