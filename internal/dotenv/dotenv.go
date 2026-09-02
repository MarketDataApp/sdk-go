// Package dotenv parses .env files into a map for the SDK's configuration
// cascade. It deliberately never mutates the process environment: a library
// must not surprise its host program (or other libraries in it) with
// variables that appear out of nowhere depending on the working directory.
package dotenv

import (
	"bufio"
	"os"
	"strings"
)

// maxLineBytes bounds a single .env line (1 MiB). Lines beyond this make
// Parse return the entries collected so far together with the error.
const maxLineBytes = 1 << 20

// Parse reads a .env file and returns its key/value pairs. It supports the
// common .env dialect: KEY=value lines, optional "export " prefixes, blank
// and #-comment lines, one balanced pair of surrounding quotes (single or
// double), and inline " #" comments on unquoted values. It never touches
// the process environment.
//
// On an oversized line Parse returns the entries parsed up to that point
// alongside the scanner error, so a broken tail cannot silently drop the
// rest of the file.
func Parse(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err // caller decides; a missing file is normal
	}
	defer func() { _ = f.Close() }()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "export"); ok && len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
			line = strings.TrimLeft(rest, " \t")
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.ContainsAny(key, " \t") {
			continue // not a plausible variable name (stray prose, etc.)
		}
		vars[key] = parseValue(strings.TrimSpace(value))
	}
	return vars, scanner.Err()
}

// parseValue strips exactly one balanced pair of surrounding quotes —
// preserving any quotes inside — or, for unquoted values, cuts an inline
// comment (a '#' preceded by whitespace).
func parseValue(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') {
		quote := v[0]
		// The closing quote is the next occurrence of the same quote
		// character (no escaping supported); anything after it — including
		// a trailing inline comment like `"value" # comment` — is discarded
		// rather than defeating the quote match by requiring it at the very
		// end of the line.
		if end := strings.IndexByte(v[1:], quote); end >= 0 {
			return v[1 : end+1]
		}
	}
	for i := 1; i < len(v); i++ {
		if v[i] == '#' && (v[i-1] == ' ' || v[i-1] == '\t') {
			return strings.TrimSpace(v[:i])
		}
	}
	return v
}
