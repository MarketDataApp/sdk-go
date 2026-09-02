package marketdata_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryWireResponseDeclaresRequiredColumns enforces that every wire
// response type implements http.ColumnRequirer.
//
// A "columns" filter makes the API return only the named columns, and each
// wire type keys its conversion off one array the caller never thinks to
// name. Miss it and a present, billed row decodes to nothing: the typed
// method reports not-found for data that is there — the same class of
// silent wrong answer the dropped "s" envelope produced, and one no unit
// test catches, because the mocked body always carries every field.
//
// Go cannot enforce this at compile time: the destination reaches the HTTP
// client as `any`, so a type that forgets the method simply falls through
// the interface check and loses its column again. A hand-kept list would go
// stale like the examples' canonicalMethods did, so the set is derived from
// the source — add a wire response type without the method and this fails.
func TestEveryWireResponseDeclaresRequiredColumns(t *testing.T) {
	for _, pkg := range []string{"stocks", "options", "funds", "markets", "utilities"} {
		t.Run(pkg, func(t *testing.T) {
			fset := token.NewFileSet()
			pkgs, err := parser.ParseDir(fset, filepath.Join(".", pkg), nil, 0)
			if err != nil {
				t.Fatalf("parsing %s: %v", pkg, err)
			}

			declared := map[string]bool{}
			hasMethod := map[string]bool{}
			for name, p := range pkgs {
				if strings.HasSuffix(name, "_test") {
					continue
				}
				for _, file := range p.Files {
					for _, decl := range file.Decls {
						switch d := decl.(type) {
						case *ast.GenDecl:
							for _, spec := range d.Specs {
								ts, ok := spec.(*ast.TypeSpec)
								if ok && strings.HasSuffix(ts.Name.Name, "Response") {
									declared[ts.Name.Name] = true
								}
							}
						case *ast.FuncDecl:
							if d.Name.Name == "RequiredColumns" && d.Recv != nil {
								hasMethod[receiverType(d)] = true
							}
						}
					}
				}
			}

			if len(declared) == 0 {
				t.Fatalf("found no wire response types in %s; the parser is not seeing the package", pkg)
			}
			for name := range declared {
				if !hasMethod[name] {
					t.Errorf("%s.%s has no RequiredColumns method, so a WithColumns caller loses the column its decode keys off (see http.ColumnRequirer)", pkg, name)
				}
			}
		})
	}
}

// receiverType names a method's receiver type, dereferencing a pointer.
func receiverType(fn *ast.FuncDecl) string {
	if len(fn.Recv.List) == 0 {
		return ""
	}
	expr := fn.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}
