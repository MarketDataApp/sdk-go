#!/bin/bash

# Count lines of code excluding single-line comments, block comments, and blank lines in Go files.

# Function to filter and count lines
count_lines() {
    local file=$1
    # Change to the root directory of the git repository
    pushd "$(git rev-parse --show-toplevel)" > /dev/null
    # Remove single-line comments, block comments, and blank lines, then count
    cat "$file" | 
    sed '/\/\*/,/\*\//d' | # Remove block comments
    grep -vE '^\s*($|//)' | # Remove blank lines and single-line comments
    wc -l | awk '{print $1}' # Print line count
    # Return to the original directory
    popd > /dev/null
}

export -f count_lines

# Find all Go files, exclude comments and blank lines, then sum the lines
total=0
for file in $(git -C "$(git rev-parse --show-toplevel)" ls-files '*.go'); do
    lines=$(count_lines "$file")
    total=$((total + lines))
done

echo "Total lines of code (excluding comments and blank lines): $total"

# Path to the README file
README_PATH="$(git rev-parse --show-toplevel)/README.md"

# Determine OS and set sed command accordingly
OS="$(uname)"
SED_CMD=""

if [ "$OS" == "Linux" ]; then
    SED_CMD="sed -i"
elif [ "$OS" == "Darwin" ]; then
    SED_CMD="sed -i ''"
else
    echo "Unsupported OS"
    exit 1
fi

# Update the LOC badge in the README.md
$SED_CMD "s/lines_of_code-[0-9]\+/lines_of_code-${total}/" "$README_PATH"

echo "Updated LOC in README.md to ${total}"