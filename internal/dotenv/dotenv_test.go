package dotenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// parseFile writes content to a temp .env and parses it.
func parseFile(t *testing.T, content string) (map[string]string, error) {
	t.Helper()
	envFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("write temp .env: %v", err)
	}
	return Parse(envFile)
}

func TestParse_Table(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{"basic", "KEY=hello\n", map[string]string{"KEY": "hello"}},
		{"comments and blanks", "# comment\nKEY=value\n\n# other\n", map[string]string{"KEY": "value"}},
		{"double quotes", `KEY="double quoted"` + "\n", map[string]string{"KEY": "double quoted"}},
		{"single quotes", "KEY='single quoted'\n", map[string]string{"KEY": "single quoted"}},
		{"nested quotes preserved", `KEY="say 'hi'"` + "\n", map[string]string{"KEY": "say 'hi'"}},
		{"export prefix", "export KEY=value\n", map[string]string{"KEY": "value"}},
		{"export prefix with tab", "export\tKEY=value\n", map[string]string{"KEY": "value"}},
		{"inline comment stripped", "KEY=value # comment\n", map[string]string{"KEY": "value"}},
		{"quoted value with trailing comment", `KEY="value" # comment` + "\n", map[string]string{"KEY": "value"}},
		{"hash inside quotes preserved", `KEY="value # not a comment"` + "\n", map[string]string{"KEY": "value # not a comment"}},
		{"hash glued to value preserved", "KEY=value#fragment\n", map[string]string{"KEY": "value#fragment"}},
		{"whitespace trimming", "  KEY  =  spaced  \n", map[string]string{"KEY": "spaced"}},
		{"invalid lines skipped", "noequalssign\nprose with spaces=x\nKEY=works\n", map[string]string{"KEY": "works"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFile(t, tc.content)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("Parse() = %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("Parse()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestParse_DoesNotTouchProcessEnv(t *testing.T) {
	_ = os.Unsetenv("TEST_DOTENV_SENTINEL")
	got, err := parseFile(t, "TEST_DOTENV_SENTINEL=leaked\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got["TEST_DOTENV_SENTINEL"] != "leaked" {
		t.Fatalf("Parse() missed the entry: %v", got)
	}
	if v, ok := os.LookupEnv("TEST_DOTENV_SENTINEL"); ok {
		t.Errorf("Parse() set process env TEST_DOTENV_SENTINEL=%q; must never touch the environment", v)
	}
}

func TestParse_LongLineSurvivors(t *testing.T) {
	// A 100 KiB value (beyond the old 64 KiB scanner default) must parse,
	// and entries after it must survive.
	long := strings.Repeat("x", 100*1024)
	got, err := parseFile(t, "BEFORE=1\nLONG="+long+"\nAFTER=2\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got["BEFORE"] != "1" || got["AFTER"] != "2" {
		t.Errorf("entries around the long line lost: BEFORE=%q AFTER=%q", got["BEFORE"], got["AFTER"])
	}
	if len(got["LONG"]) != 100*1024 {
		t.Errorf("LONG length = %d, want %d", len(got["LONG"]), 100*1024)
	}
}

func TestParse_OversizedLineReturnsPartialAndError(t *testing.T) {
	// Beyond the 1 MiB hard cap: the entries before the oversized line must
	// still be returned, alongside the scanner error.
	huge := strings.Repeat("x", maxLineBytes+1)
	got, err := parseFile(t, "BEFORE=1\nHUGE="+huge+"\n")
	if err == nil {
		t.Fatal("Parse() with an oversized line should return an error")
	}
	if got["BEFORE"] != "1" {
		t.Errorf("entries before the oversized line lost: %v", got)
	}
}

func TestParse_FileNotFound(t *testing.T) {
	if _, err := Parse("/nonexistent/path/.env"); err == nil {
		t.Fatal("Parse() should return error for nonexistent file")
	}
}
