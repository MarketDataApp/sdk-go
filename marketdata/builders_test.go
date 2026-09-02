package marketdata_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryOptionTakingMethodUsesABuilder enforces the invariant: a
// request is serialized in exactly one place per endpoint, the
// shared *Path builder that the JSON method and both formatted facets all
// consume. A second serializer is how a facet silently drops an option the
// JSON method sends.
//
// Gap 64 recorded that invariant as "impossible by construction", but
// nothing enforced it, and it was false: Service.Candles kept its own copy
// of the validation and the chunk-split decision for the whole release,
// bound to the builder only by a comment, and the two had already drifted
// textually before the PR #33 round-2 review found them. Construction does
// not hold itself; this test does.
//
// The rule needs no allowlist. A method that accepts functional options has
// something to serialize and must delegate that to a builder; the five that
// take none (options.Lookup, options.LookupQuery and the three utilities
// methods) address a fixed path with no parameters at all.
func TestEveryOptionTakingMethodUsesABuilder(t *testing.T) {
	for _, pkg := range []string{"stocks", "options", "funds", "markets", "utilities"} {
		t.Run(pkg, func(t *testing.T) {
			fset := token.NewFileSet()
			pkgs, err := parser.ParseDir(fset, filepath.Join(".", pkg), nil, 0)
			if err != nil {
				t.Fatalf("parsing %s: %v", pkg, err)
			}

			// calls maps a declaration to the names it calls. Methods are
			// keyed by receiver type ("Service.Candles"), plain functions by
			// name. Keying by bare name instead makes the whole check
			// vacuous: CSVService.Candles and htmlService.Candles share the
			// name with the JSON method and both DO call candlesPath, so a
			// single namespace reports the JSON method as compliant no
			// matter what it does. Verified: keyed by bare name this test
			// passes against the pre-fix commit.
			calls := map[string][]string{}
			var entryPoints []string
			for name, p := range pkgs {
				if strings.HasSuffix(name, "_test") {
					continue
				}
				for _, file := range p.Files {
					for _, decl := range file.Decls {
						fn, ok := decl.(*ast.FuncDecl)
						if !ok {
							continue
						}
						key := declKey(fn)
						calls[key] = append(calls[key], calledNames(fn)...)
						if fn.Recv != nil && isServiceMethod(fn) && fn.Name.IsExported() && takesContext(fn) && takesOptions(fn) {
							entryPoints = append(entryPoints, fn.Name.Name)
						}
					}
				}
			}

			if len(entryPoints) == 0 {
				if pkg == "utilities" {
					return // no method here takes options; nothing to check
				}
				t.Fatalf("found no option-taking methods in %s; the parser is not seeing the package", pkg)
			}

			for _, entry := range entryPoints {
				if !reachesBuilder("Service."+entry, calls, map[string]bool{}) {
					t.Errorf("%s.Service.%s accepts options but reaches no *Path builder, so it serializes its own request — the second-serializer shape this guard exists to prevent", pkg, entry)
				}
			}
		})
	}
}

// takesOptions reports whether fn's last parameter is a variadic ...Option.
func takesOptions(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	last := fn.Type.Params.List[len(fn.Type.Params.List)-1]
	ellipsis, ok := last.Type.(*ast.Ellipsis)
	if !ok {
		return false
	}
	ident, ok := ellipsis.Elt.(*ast.Ident)
	return ok && strings.HasSuffix(ident.Name, "Option")
}

// calledNames lists the function and method names fn calls, by their bare
// name — quotePath(...) yields "quotePath", s.quoteRaw(...) yields
// "quoteRaw".
func calledNames(fn *ast.FuncDecl) []string {
	var names []string
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch f := call.Fun.(type) {
		case *ast.Ident:
			names = append(names, f.Name)
		case *ast.SelectorExpr:
			names = append(names, f.Sel.Name)
		}
		return true
	})
	return names
}

// declKey names a declaration for the call graph: a method by its receiver
// type, a plain function by its own name.
func declKey(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return fn.Name.Name
	}
	expr := fn.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name + "." + fn.Name.Name
	}
	return fn.Name.Name
}

// reachesBuilder reports whether key transitively calls a *Path builder,
// following delegation through the package's own helpers (options.Quotes
// reaches quotePath only via quoteEach). A callee is resolved as a Service
// method first and as a plain function second, since a bare selector name
// carries no receiver type.
func reachesBuilder(key string, calls map[string][]string, seen map[string]bool) bool {
	if seen[key] {
		return false
	}
	seen[key] = true
	for _, callee := range calls[key] {
		if strings.HasSuffix(callee, "Path") {
			return true
		}
		if reachesBuilder("Service."+callee, calls, seen) || reachesBuilder(callee, calls, seen) {
			return true
		}
	}
	return false
}
