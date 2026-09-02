// Package examples_test hosts cross-cutting checks over the examples tree
// that don't belong to any single example module. It has no non-test
// counterpart package (examples/ has no root-module .go files of its own),
// so it never enters the root module's coverage gate.
package examples_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// fetchFiles are the TUI apps' fetch layers, the sole home for every SDK
// call each app makes: every SDK call lives in fetch.go, one function per
// operation. Paths are relative to this test file's own
// directory (examples/), independent of the caller's working directory.
var fetchFiles = []string{
	"stockterm/fetch.go",
	"optionterm/fetch.go",
}

// selectorPattern matches a "client.<Service>.<Method>(" call site. The
// method-name class ([A-Za-z0-9_]+) is greedy and anchored on the "(" that
// follows it, so it always captures the longest possible identifier before
// the call parenthesis. That is what keeps prefix collisions apart: against
// "client.Markets.StatusHistory(" the match captures "StatusHistory" in
// full, never backtracking to stop at "Status", so Markets.Status and
// Markets.StatusHistory can never be confused for one another.
var selectorPattern = regexp.MustCompile(`client\.(Stocks|Options|Funds|Markets|Utilities)\.([A-Za-z0-9_]+)\(`)

// exemptMethods are context-first SDK service methods the two TUI apps
// deliberately do not call, each with the reason. Everything NOT listed here
// must be exercised by one of them.
//
// The canonical set used to be a hand-written list of 18 names. It froze:
// the SDK grew to 21 context-first methods and the three below were in
// neither the list nor either app, so the guard — whose whole purpose is
// noticing an app losing coverage — was structurally incapable of seeing a
// method it had never been told about. The set is now derived from the SDK
// source, which turns "silently absent" into "must be explicitly decided".
var exemptMethods = map[string]string{
	"Options.QuoteHistory":   "a historical window for one contract; optionterm shows the live chain and a pinned contract, neither of which is a time series",
	"Options.QuotesBySymbol": "a batch keyed by underlying; optionterm works one underlying at a time, so the batch form has no screen",
	"Options.LookupQuery":    "free-form contract lookup; optionterm's [/] lookup uses the typed Lookup, which is the form its input fields collect",
}

// sdkMethods returns every context-first service method the SDK exposes,
// derived from the source rather than listed by hand.
func sdkMethods(t *testing.T) map[string]bool {
	t.Helper()

	out := map[string]bool{}
	for _, pkg := range []string{"stocks", "options", "funds", "markets", "utilities"} {
		dir := filepath.Join("..", "marketdata", pkg)
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", dir, err)
		}
		service := strings.ToUpper(pkg[:1]) + pkg[1:]
		for name, p := range pkgs {
			if strings.HasSuffix(name, "_test") {
				continue
			}
			for _, file := range p.Files {
				for _, decl := range file.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Recv == nil || !fn.Name.IsExported() {
						continue
					}
					if strings.HasPrefix(fn.Name.Name, "Get") || !onService(fn) || !contextFirst(fn) {
						continue
					}
					out[service+"."+fn.Name.Name] = true
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("derived no SDK methods; the parser is not seeing marketdata/")
	}
	return out
}

func onService(fn *ast.FuncDecl) bool {
	if len(fn.Recv.List) != 1 {
		return false
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	ident, ok := star.X.(*ast.Ident)
	return ok && ident.Name == "Service"
}

func contextFirst(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	sel, ok := fn.Type.Params.List[0].Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	return ok && pkgIdent.Name == "context" && sel.Sel.Name == "Context"
}

// TestSDKMethodCoverage enforces the reference apps' coverage criterion:
// stockterm and
// optionterm, together, must exercise every one of the 18 context-first SDK
// service methods listed in canonicalMethods (the plan's "Canonical SDK
// coverage list" section, which also documents the App column recording
// which app is responsible for each method).
//
// It parses fetchFiles for "client.<Service>.<Method>(" selectors, unions
// the set found across both files, and fails the build in two directions:
//
//   - MISSING: a canonical method that neither fetch.go calls. This is the
//     primary regression this test guards against — an app losing coverage
//     of an SDK method during a refactor.
//   - UNKNOWN: a selector found in a fetch.go that isn't in canonicalMethods.
//     This catches drift the other way — a method renamed in one place but
//     not the other, or a new call added without updating canonicalMethods
//     (and, in turn, the plan's coverage table) to match.
func TestSDKMethodCoverage(t *testing.T) {
	canonicalMethods := sdkMethods(t)
	found := make(map[string]bool)

	for _, path := range fetchFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		for _, m := range selectorPattern.FindAllStringSubmatch(string(data), -1) {
			found[m[1]+"."+m[2]] = true
		}
	}

	var missing []string
	for method := range canonicalMethods {
		if !found[method] && exemptMethods[method] == "" {
			missing = append(missing, method)
		}
	}
	sort.Strings(missing)

	// A stale exemption is its own failure: it must still name a method the
	// SDK has, and one the apps still do not call.
	var staleExemptions []string
	for method, reason := range exemptMethods {
		switch {
		case !canonicalMethods[method]:
			staleExemptions = append(staleExemptions, method+" (no such SDK method)")
		case found[method]:
			staleExemptions = append(staleExemptions, method+" (an app now calls it; drop the exemption: "+reason+")")
		}
	}
	sort.Strings(staleExemptions)
	if len(staleExemptions) > 0 {
		t.Errorf("stale exemptions in exemptMethods: %s", strings.Join(staleExemptions, ", "))
	}

	var unknown []string
	for selector := range found {
		if !canonicalMethods[selector] && exemptMethods[selector] == "" {
			unknown = append(unknown, selector)
		}
	}
	sort.Strings(unknown)

	if len(missing) > 0 {
		t.Errorf("missing SDK method coverage across %s: %s",
			strings.Join(fetchFiles, ", "), strings.Join(missing, ", "))
	}
	if len(unknown) > 0 {
		t.Errorf("unknown selector(s) not in the canonical 18-method set (rename drift?): %s",
			strings.Join(unknown, ", "))
	}
}
