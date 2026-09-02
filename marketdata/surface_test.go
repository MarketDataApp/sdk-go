package marketdata_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryServiceMethodHasGetForm enforces the promise marketdata/doc.go
// makes on pkg.go.dev: "Every service method has two forms: a context-first
// form ... and a Get-prefixed convenience form."
//
// It was false for BulkCandles and StatusHistory, and nothing recorded the
// omission as deliberate — unlike the CSV facet's exclusions, which ADR-018
// fixes explicitly. A hand-written list would go stale the same way the
// examples' canonicalMethods did, so the pair is derived from the source:
// add a context-first method without its wrapper and this fails.
func TestEveryServiceMethodHasGetForm(t *testing.T) {
	for _, pkg := range []string{"stocks", "options", "funds", "markets", "utilities"} {
		t.Run(pkg, func(t *testing.T) {
			fset := token.NewFileSet()
			pkgs, err := parser.ParseDir(fset, filepath.Join(".", pkg), nil, 0)
			if err != nil {
				t.Fatalf("parsing %s: %v", pkg, err)
			}

			contextFirst := map[string]bool{}
			getForm := map[string]bool{}
			for name, p := range pkgs {
				if strings.HasSuffix(name, "_test") {
					continue
				}
				for _, file := range p.Files {
					for _, decl := range file.Decls {
						fn, ok := decl.(*ast.FuncDecl)
						if !ok || fn.Recv == nil || !fn.Name.IsExported() || !isServiceMethod(fn) {
							continue
						}
						if after, found := strings.CutPrefix(fn.Name.Name, "Get"); found && after != "" {
							getForm[after] = true
							continue
						}
						if takesContext(fn) {
							contextFirst[fn.Name.Name] = true
						}
					}
				}
			}

			if len(contextFirst) == 0 {
				t.Fatalf("found no context-first methods in %s; the parser is not seeing the package", pkg)
			}
			for name := range contextFirst {
				if !getForm[name] {
					t.Errorf("%s.Service.%s has no Get%s convenience form, which marketdata/doc.go promises for every service method", pkg, name, name)
				}
			}
		})
	}
}

// isServiceMethod reports whether fn is a method on *Service — excluding the
// CSV and HTML facets, whose endpoint sets ADR-018 fixes deliberately.
func isServiceMethod(fn *ast.FuncDecl) bool {
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

func takesContext(fn *ast.FuncDecl) bool {
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
