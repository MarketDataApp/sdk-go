#!/bin/bash

# Function to create a new temporary directory
create_tmp_dir() {
    echo $(mktemp -d)
}

# Adjust to point to the project root from .scripts
SRC_DIR="$(dirname "$0")/.."
# Define the output directory for the markdown file within the .scripts directory
OUTPUT_DIR="$(dirname "$0")"
MAIN_FILE="indices_candles.go" # Main source file
TEST_FILE="indices_candles_test.go" # Corresponding test file
MODELS_DIR="models" # Models directory relative to the project root
MODEL_MAIN_FILE="$MODELS_DIR/indices_candles.go" # Main model file
MODEL_TEST_FILE="$MODELS_DIR/indices_candles_test.go" # Model test file

# Step 1: Documentation for main files
TMP_DIR=$(create_tmp_dir)
echo "Using temporary directory for main files: $TMP_DIR"

# Copy the main file and its test file to the temporary directory
cp "$SRC_DIR/$MAIN_FILE" "$TMP_DIR"
cp "$SRC_DIR/$TEST_FILE" "$TMP_DIR"

# Run gomarkdoc on the temporary directory for main files
gomarkdoc --output "$OUTPUT_DIR/indices_candles_request.md" "$TMP_DIR"

# Clean up: remove the temporary directory for main files
rm -rf "$TMP_DIR"

# Step 2: Documentation for model files
TMP_DIR=$(create_tmp_dir)
echo "Using new temporary directory for model files: $TMP_DIR"

# Copy the model file and its test file to the new temporary directory
cp "$SRC_DIR/$MODEL_MAIN_FILE" "$TMP_DIR"
cp "$SRC_DIR/$MODEL_TEST_FILE" "$TMP_DIR"

# Run gomarkdoc on the new temporary directory for model files
gomarkdoc --output "$OUTPUT_DIR/indices_candles_response.md" "$TMP_DIR"

# Clean up: remove the new temporary directory for model files
rm -rf "$TMP_DIR"

echo "Documentation generated for main files and model files"