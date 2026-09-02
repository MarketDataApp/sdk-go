package apicatalog

import "testing"

func TestKindString(t *testing.T) {
	cases := map[Kind]string{Query: "query", Path: "path", Residual: "residual", Kind(99): "unknown"}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("Kind(%d).String() = %q, want %q", k, got, want)
		}
	}
}

// TestCatalogInvariants sanity-checks the catalog data itself: every entry has
// an endpoint and a name, and every parameter with a non-empty exclusivity
// group shares that group with at least one other parameter on the same
// endpoint (a group of one is a data error).
func TestCatalogInvariants(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("catalog is empty")
	}
	type key struct{ endpoint, group string }
	groupCount := map[key]int{}
	for _, p := range all {
		if p.Endpoint == "" || p.Name == "" {
			t.Errorf("entry with empty endpoint/name: %+v", p)
		}
		if p.SDKPath == "" {
			t.Errorf("entry %s.%s has no SDK path", p.Endpoint, p.Name)
		}
		if p.Group != "" {
			groupCount[key{p.Endpoint, p.Group}]++
		}
	}
	for k, n := range groupCount {
		if n < 2 {
			t.Errorf("exclusivity group %q on %s has only %d member (expected >= 2)", k.group, k.endpoint, n)
		}
	}
}
