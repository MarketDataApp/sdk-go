#!/bin/bash

set -e # Exit immediately if a command exits with a non-zero status.

# Function to create a new temporary directory
create_tmp_dir() {
    echo $(mktemp -d)
}

# Adjust to point to the project root from .scripts
SRC_DIR="$(dirname "$0")/.."
OUTPUT_DIR="$(dirname "$0")"
MODELS_DIR="models" # Models directory relative to the project root

# Define the destination directory to Docusaurus
DEST_DIR="${SRC_DIR}/../documentation/sdk/go/"

# Function to move and merge the 'go' directory to the destination directory
move_and_merge_go_dir() {
    SOURCE_DIR="$OUTPUT_DIR/go" # Assuming the 'go' folder is directly inside the OUTPUT_DIR

    # Use rsync to copy files from source to destination. 
    rsync -av --ignore-existing --remove-source-files "$SOURCE_DIR/" "$DEST_DIR/"

    # Find and remove empty directories in the source directory
    find "$SOURCE_DIR" -type d -empty -delete

    # After running rsync and find
    if [ -z "$(ls -A "$SOURCE_DIR")" ]; then
        echo "$SOURCE_DIR is empty, removing..."
        rmdir "$SOURCE_DIR"
    fi

    echo "Moved and merged 'go' directory from $SOURCE_DIR to $DEST_DIR"
}

# Function to process a group of files based on the group name
process_group() {
    GROUP_NAME=$1

    MAIN_FILE="${GROUP_NAME}.go"
    TEST_FILE="${GROUP_NAME}_test.go"
    MODEL_MAIN_FILE="$MODELS_DIR/${GROUP_NAME}.go"
    MODEL_TEST_FILE="$MODELS_DIR/${GROUP_NAME}_test.go"
    CANDLE_FILE="$MODELS_DIR/candle.go" # Path to the candle.go file
    CANDLE_TEST_FILE="$MODELS_DIR/candle_test.go" # Path to the candle.go file


    # Documentation for main files
    TMP_DIR=$(create_tmp_dir)
    echo "Using temporary directory for main files: $TMP_DIR"

    if [ -f "$SRC_DIR/$MAIN_FILE" ]; then
        cp "$SRC_DIR/$MAIN_FILE" "$TMP_DIR"
    fi
    if [ -f "$SRC_DIR/$TEST_FILE" ]; then
        cp "$SRC_DIR/$TEST_FILE" "$TMP_DIR"
    fi
    gomarkdoc --output "$OUTPUT_DIR/${GROUP_NAME}_request.md" "$TMP_DIR"

    rm -rf "$TMP_DIR"

    # Documentation for model files
    TMP_DIR=$(create_tmp_dir)
    echo "Using new temporary directory for model files: $TMP_DIR"

    if [ -f "$SRC_DIR/$MODEL_MAIN_FILE" ]; then
        cp "$SRC_DIR/$MODEL_MAIN_FILE" "$TMP_DIR"
    fi
    if [ -f "$SRC_DIR/$MODEL_TEST_FILE" ]; then
        cp "$SRC_DIR/$MODEL_TEST_FILE" "$TMP_DIR"
    fi
    # Check if the group name contains "candles" and copy candle.go if it does
    if [[ "$GROUP_NAME" == *"candles"* ]]; then
        echo "Group name contains 'candles', copying candles files as well"
        if [ -f "$SRC_DIR/$CANDLE_FILE" ]; then
            cp "$SRC_DIR/$CANDLE_FILE" "$TMP_DIR"
        fi
        if [ -f "$SRC_DIR/$CANDLE_TEST_FILE" ]; then
            cp "$SRC_DIR/$CANDLE_TEST_FILE" "$TMP_DIR"
        fi
    fi

    gomarkdoc --output "$OUTPUT_DIR/${GROUP_NAME}_response.md" "$TMP_DIR"

    # Clean up temporary directory
    rm -rf "$TMP_DIR"

    # Run the Python script on all markdown files
    "$OUTPUT_DIR/process_markdown.py" "$OUTPUT_DIR"/*.md

    # Remove the Markdown files
    rm "$OUTPUT_DIR"/*.md

    echo "Markdown processing and cleanup completed for $GROUP_NAME"
}

# Call process_group for each group name
process_group "indices_candles"
process_group "indices_quotes"

# Add more calls to process_group with different group names as needed

# After all process_group calls, move and merge the 'go' directory
move_and_merge_go_dir
