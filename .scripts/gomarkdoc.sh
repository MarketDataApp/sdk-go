#!/bin/bash

set -e # Exit immediately if a command exits with a non-zero status.

# Flag to determine whether cleanup should be performed
PERFORM_CLEANUP=true

# Parse command-line arguments
for arg in "$@"
do
    case $arg in
        --no-cleanup)
        PERFORM_CLEANUP=false
        shift # Remove --no-cleanup from processing
        ;;
        *)
        # Unknown option
        ;;
    esac
done

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
    if [ "$PERFORM_CLEANUP" = true ]; then
        # Use rsync to copy files from source to destination and remove the source files.
        rsync -av --remove-source-files "$SOURCE_DIR/" "$DEST_DIR/"
    else
        # Use rsync to copy files from source to destination without removing the source files.
        rsync -av "$SOURCE_DIR/" "$DEST_DIR/"
    fi
    # Find and remove empty directories in the source directory
    find "$SOURCE_DIR" -type d -empty -delete

    echo "Moved and merged 'go' directory from $SOURCE_DIR to $DEST_DIR"
}

# Function to process a group of files based on the group name
process_group() {
    GROUP_NAME=$1

    # Special handling for the "options_chain" group to use "options_quotes.go" for models
    if [ "$GROUP_NAME" == "options_chain" ]; then
        echo "Processing options_chain group with an exception"
        MODEL_MAIN_FILE="$MODELS_DIR/options_quotes.go"
        MODEL_TEST_FILE="$MODELS_DIR/options_quotes_test.go"
    else
        # Remove the word "bulk" from the group name if it exists
        MODEL_GROUP_NAME=${GROUP_NAME/bulk/}

        MODEL_MAIN_FILE="$MODELS_DIR/${MODEL_GROUP_NAME}.go"
        MODEL_TEST_FILE="$MODELS_DIR/${MODEL_GROUP_NAME}_test.go"
    fi

    MAIN_FILE="${GROUP_NAME}.go"
    TEST_FILE="${GROUP_NAME}_test.go"
    CANDLE_FILE="$MODELS_DIR/candle.go" # Path to the candle.go file
    CANDLE_TEST_FILE="$MODELS_DIR/candle_test.go" # Path to the candle.go file


    # Documentation for main files
    TMP_DIR=$(create_tmp_dir)
    echo "Using temporary directory for main files: $TMP_DIR"

    MAIN_FILES_COPIED=false

    if [ -f "$SRC_DIR/$MAIN_FILE" ]; then
        echo "Copying file: $SRC_DIR/$MAIN_FILE to $TMP_DIR"
        cp "$SRC_DIR/$MAIN_FILE" "$TMP_DIR"
        MAIN_FILES_COPIED=true
    fi
    if [ -f "$SRC_DIR/$TEST_FILE" ]; then
        echo "Copying file: $SRC_DIR/$TEST_FILE to $TMP_DIR"
        cp "$SRC_DIR/$TEST_FILE" "$TMP_DIR"
        MAIN_FILES_COPIED=true
    fi
    if [ "$MAIN_FILES_COPIED" = true ]; then
        gomarkdoc --output "$OUTPUT_DIR/${GROUP_NAME}_request.md" "$TMP_DIR"
    fi

    rm -rf "$TMP_DIR"

    # Documentation for model files
    TMP_DIR=$(create_tmp_dir)
    echo "Using new temporary directory for model files: $TMP_DIR"

    MODEL_FILES_COPIED=false

    if [ -f "$SRC_DIR/$MODEL_MAIN_FILE" ]; then
        echo "Copying file: $SRC_DIR/$MODEL_MAIN_FILE to $TMP_DIR"
        cp "$SRC_DIR/$MODEL_MAIN_FILE" "$TMP_DIR"
        MODEL_FILES_COPIED=true
    fi
    if [ -f "$SRC_DIR/$MODEL_TEST_FILE" ]; then
        echo "Copying file: $SRC_DIR/$MODEL_TEST_FILE to $TMP_DIR"
        cp "$SRC_DIR/$MODEL_TEST_FILE" "$TMP_DIR"
        MODEL_FILES_COPIED=true
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

    if [ "$MODEL_FILES_COPIED" = true ]; then
        gomarkdoc --output "$OUTPUT_DIR/${GROUP_NAME}_response.md" "$TMP_DIR"
    fi

    # Clean up temporary directory
    rm -rf "$TMP_DIR"

    # Run the Python script on all markdown files
    "$OUTPUT_DIR/process_markdown.py" "$OUTPUT_DIR"/*.md


    if [ "$PERFORM_CLEANUP" = true ]; then
        # Remove the Markdown files
        rm "$OUTPUT_DIR"/*.md
    fi

    echo "Markdown processing and cleanup completed for $GROUP_NAME"
}

# Call process_group for each group name
process_group "indices_candles"
process_group "indices_quotes"
process_group "markets_status"
process_group "stocks_candles"
process_group "stocks_quotes"
process_group "stocks_earnings"
process_group "stocks_news"
process_group "stocks_bulkcandles"
process_group "stocks_bulkquotes"
process_group "options_expirations"
process_group "options_lookup"
process_group "options_quotes"
process_group "options_strikes"
process_group "options_chain"
process_group "client"



# Add more calls to process_group with different group names as needed

# After all process_group calls, move and merge the 'go' directory
move_and_merge_go_dir
