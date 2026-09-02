package options

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestExpirationTypeFilter_Constructors(t *testing.T) {
	inc := IncludeExpirationTypes(Weekly, Monthly)
	include, types := inc.expirationTypes()
	if !include || len(types) != 2 || types[0] != Weekly || types[1] != Monthly {
		t.Errorf("include filter = (%v, %v), want (true, [weekly monthly])", include, types)
	}
	exc := ExcludeExpirationTypes(Quarterly)
	include, types = exc.expirationTypes()
	if include || len(types) != 1 || types[0] != Quarterly {
		t.Errorf("exclude filter = (%v, %v), want (false, [quarterly])", include, types)
	}

	opts := defaultChainOptions()
	WithExpirationTypes(inc).apply(opts)
	if opts.expTypes == nil {
		t.Fatal("WithExpirationTypes should store the filter")
	}
}

// TestChain_ExpirationTypesInclude proves IncludeExpirationTypes sends each
// named cadence as true on the wire.
func TestChain_ExpirationTypesInclude(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("weekly") != "true" || q.Get("monthly") != "true" {
			t.Errorf("weekly=%q monthly=%q, want both true", q.Get("weekly"), q.Get("monthly"))
		}
		if q.Get("quarterly") != "" {
			t.Errorf("quarterly=%q, want unset", q.Get("quarterly"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(quoteResponse{Status: "ok", OptionSymbol: []string{}})
	})
	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithExpirationTypes(IncludeExpirationTypes(Weekly, Monthly)))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}
