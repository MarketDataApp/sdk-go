#!/bin/bash

# Count lines of code excluding single-line comments, block comments, and blank lines in Go files.

count_lines() {
    local file="$1"
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
echo "README_PATH is set to: $README_PATH"

# Determine OS and set sed command accordingly
OS="$(uname)"
echo "Operating System: $OS"

# Update the LOC badge in the README.md
echo "Attempting to update LOC in README.md..."
if [ "$OS" == "Darwin" ]; then
    # Ensure the pattern matches exactly what's in your README.md, including any fixed parts of the badge URL
    sed -i '' "s/lines_of_code-[0-9]\{1,\}/lines_of_code-${total}/" "$README_PATH" && echo "Update attempt made."
else
    echo "This script currently supports updates on Darwin (macOS) only."
fi

# Verification step
if grep -q "lines_of_code-${total}" "$README_PATH"; then
    echo "Verification: README.md successfully updated."
else
    echo "Verification: Update to README.md failed or no change needed."
fi