//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"
)

// The offline audit in internal/apicatalog/openapi_test.go compares the SDK's
// parameter catalog against a checked-in copy of the API's published OpenAPI
// schema. That comparison is only as good as the copy: once the fixture drifts
// behind the live API, the audit keeps passing while the thing it audits has
// moved.
//
// This test is what keeps the fixture honest. It fetches the live schema and
// fails when it no longer matches, printing the exact difference — a new
// endpoint, a new parameter, or one that was removed. It runs in the
// integration suite, which CI executes on every pull request.
//
// To adopt the live schema after reviewing the difference:
//
//	MARKETDATA_UPDATE_OPENAPI=1 go test ./integration/... -tags=integration \
//	    -run TestOpenAPI_SchemaMatchesFixture
//
// That rewrites the fixture in place. Review the diff: a parameter appearing
// there is a capability the SDK may now need to expose, and the offline audit
// will fail until it is either implemented or allowlisted with a reason.
//
// When running locally, pass -count=1. This test's verdict depends on a remote
// document, but `go test` caches by input files and flags and will happily
// replay a previous PASS without fetching anything. CI is unaffected: the
// integration job sets `cache: false`, so every run starts with an empty build
// cache.

const (
	openapiSchemaURL = "https://api.marketdata.app/schema/?format=json"
	// TestMain chdirs the process to the module root, so this path is
	// relative to the repository root rather than to this package.
	openapiFixtureRel = "internal/apicatalog/testdata/openapi_v1.json"
)

// openapiDoc is the fixture's shape, mirroring the offline test's reader.
type openapiDoc struct {
	Source  string              `json:"_source"`
	Fetched string              `json:"_fetched"`
	Note    string              `json:"_note"`
	Paths   map[string][]string `json:"paths"`
}

// liveSchema is the subset of the OpenAPI document this audit reads.
type liveSchema struct {
	Paths map[string]struct {
		Get *struct {
			Parameters []struct {
				Name string `json:"name"`
				In   string `json:"in"`
			} `json:"parameters"`
		} `json:"get"`
	} `json:"paths"`
}

// fetchLiveSchemaPaths returns the query parameters the live schema declares
// per GET path, in the fixture's normalized form.
func fetchLiveSchemaPaths(t *testing.T) map[string][]string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openapiSchemaURL, nil)
	if err != nil {
		t.Fatalf("build schema request: %v", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("fetch %s: %v", openapiSchemaURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fetch %s: HTTP %d, want 200", openapiSchemaURL, resp.StatusCode)
	}

	var doc liveSchema
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("live schema declares no paths; refusing to compare against an empty document")
	}

	out := make(map[string][]string, len(doc.Paths))
	for path, item := range doc.Paths {
		if item.Get == nil {
			continue
		}
		names := []string{}
		for _, p := range item.Get.Parameters {
			if p.In == "query" {
				names = append(names, p.Name)
			}
		}
		sort.Strings(names)
		out[path] = names
	}
	return out
}

func readOpenAPIFixture(t *testing.T) openapiDoc {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(openapiFixtureRel))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fx openapiDoc
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return fx
}

// TestOpenAPI_SchemaMatchesFixture fails when the API's published schema has
// changed since the fixture was captured.
func TestOpenAPI_SchemaMatchesFixture(t *testing.T) {
	live := fetchLiveSchemaPaths(t)

	if os.Getenv("MARKETDATA_UPDATE_OPENAPI") != "" {
		writeOpenAPIFixture(t, live)
		t.Log("fixture rewritten from the live schema; review the diff and re-run the offline audit")
		return
	}

	fx := readOpenAPIFixture(t)

	var problems []string
	for path, liveParams := range live {
		fixParams, ok := fx.Paths[path]
		if !ok {
			problems = append(problems, fmt.Sprintf(
				"NEW PATH %s (query params: %v)", path, liveParams))
			continue
		}
		if !reflect.DeepEqual(liveParams, fixParams) {
			added := difference(liveParams, fixParams)
			removed := difference(fixParams, liveParams)
			problems = append(problems, fmt.Sprintf(
				"CHANGED %s: added %v, removed %v", path, added, removed))
		}
	}
	for path := range fx.Paths {
		if _, ok := live[path]; !ok {
			problems = append(problems, fmt.Sprintf("REMOVED PATH %s", path))
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		t.Errorf("the API's published schema no longer matches testdata/openapi_v1.json "+
			"(captured %s). The SDK's parameter audit is comparing against stale data.\n\n%s\n\n"+
			"Review each line: an added parameter may be a capability the SDK should expose. "+
			"Then adopt the new schema with:\n"+
			"  MARKETDATA_UPDATE_OPENAPI=1 go test ./integration/... -tags=integration -run %s",
			fx.Fetched, joinLines(problems), t.Name())
	}
}

// writeOpenAPIFixture rewrites the fixture from the live schema, preserving the
// provenance fields and stamping today's date.
func writeOpenAPIFixture(t *testing.T, paths map[string][]string) {
	t.Helper()

	prev := readOpenAPIFixture(t)
	doc := openapiDoc{
		Source:  prev.Source,
		Fetched: time.Now().UTC().Format("2006-01-02"),
		Note:    prev.Note,
		Paths:   paths,
	}

	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Clean(openapiFixtureRel), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

// difference returns the elements of a that are not in b.
func difference(a, b []string) []string {
	in := make(map[string]bool, len(b))
	for _, s := range b {
		in[s] = true
	}
	out := []string{}
	for _, s := range a {
		if !in[s] {
			out = append(out, s)
		}
	}
	return out
}

func joinLines(ss []string) string {
	out := ""
	for _, s := range ss {
		out += "  " + s + "\n"
	}
	return out
}
