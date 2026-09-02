package apicatalog

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTrackedDiscrepancyLinksAreFiled enforces the tracked-discrepancy
// pattern's contract: every skip references a real filed issue, because the
// external tracker is what prompts the skip to be lifted when the API is
// fixed. A placeholder link is a permanent skip nobody revisits — the same
// silent-rot shape the stale-allowlist assertion in this package already
// rejects. The two discrepancies that shipped with issues/TBD links now
// point at MarketData-App/api#352 and #375; this keeps the next one from
// repeating that.
func TestTrackedDiscrepancyLinksAreFiled(t *testing.T) {
	path := filepath.Join("..", "..", "integration", "discrepancy_test.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	placeholder := regexp.MustCompile(`issues/TBD|issue TBD`)
	// sdk-go-v2 was the private, temporary home of this SDK while v2 was
	// unreleased; it is discarded once v2 ships from MarketDataApp/sdk-go.
	// A skip pointing there would be a permanently-404 link in a public
	// repo — the same silent rot as a TBD, just harder to spot, since the
	// URL looks filed. Defects in the API belong in MarketData-App/api.
	disposable := regexp.MustCompile(`MarketDataApp/sdk-go-v2`)
	for i, line := range strings.Split(string(src), "\n") {
		if placeholder.MatchString(line) {
			t.Errorf("%s:%d: tracked discrepancy references a placeholder issue: %s", path, i+1, strings.TrimSpace(line))
		}
		if disposable.MatchString(line) {
			t.Errorf("%s:%d: tracked discrepancy references the disposable sdk-go-v2 repo; file it under MarketData-App/api instead: %s", path, i+1, strings.TrimSpace(line))
		}
	}
}
