package utilities

// Wire-contract tests (ADR-010): hand-written fixtures from testdata/,
// asserted field by field. See the stocks package for the rationale.

import (
	"context"
	"math"
	"net/http"
	"os"
	"testing"
)

func fixtureHandler(t *testing.T, name string, header http.Header) http.Handler {
	t.Helper()
	body, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, vs := range header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
}

func TestWireContract_APIStatus(t *testing.T) {
	svc := newTestService(fixtureHandler(t, "status.json", nil))
	status, _, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status == nil {
		t.Fatal("nil status")
	}
	// All services online -> aggregate "online"; uptimes are wire fractions
	// averaged and converted to percentages.
	if status.Status != "online" {
		t.Errorf("Status = %q, want online", status.Status)
	}
	if math.Abs(status.Uptime30d-99.8) > 1e-9 {
		t.Errorf("Uptime30d = %v, want 99.8", status.Uptime30d)
	}
	if math.Abs(status.Uptime90d-99.4) > 1e-9 {
		t.Errorf("Uptime90d = %v, want 99.4", status.Uptime90d)
	}
	if status.Updated.Unix() != 1704067200 {
		t.Errorf("Updated = %v, want unix 1704067200", status.Updated)
	}
}

func TestWireContract_Headers(t *testing.T) {
	svc := newTestService(fixtureHandler(t, "headers.json", nil))
	headers, _, err := svc.Headers(context.Background())
	if err != nil {
		t.Fatalf("Headers() error = %v", err)
	}
	if headers == nil || len(headers.Headers) != 2 {
		t.Fatalf("headers = %+v, want 2 entries", headers)
	}
	if headers.Headers["Authorization"] != "Bearer **********1234" {
		t.Errorf("Authorization = %q, does not match fixture", headers.Headers["Authorization"])
	}
	if headers.Headers["User-Agent"] != "marketdata-sdk-go/2.0.0" {
		t.Errorf("User-Agent = %q, does not match fixture", headers.Headers["User-Agent"])
	}
}

func TestWireContract_User(t *testing.T) {
	// Credit fields come from the x-api-ratelimit-* HEADERS (body request
	// counts are the fallback); permissions come from the body.
	hdr := http.Header{}
	hdr.Set("X-Api-Ratelimit-Limit", "100000")
	hdr.Set("X-Api-Ratelimit-Remaining", "91201")
	hdr.Set("X-Api-Ratelimit-Consumed", "8799")
	hdr.Set("X-Api-Ratelimit-Reset", "1704114000")
	svc := newTestService(fixtureHandler(t, "user.json", hdr))
	user, _, err := svc.User(context.Background())
	if err != nil {
		t.Fatalf("User() error = %v", err)
	}
	if user == nil {
		t.Fatal("nil user")
	}
	if user.CreditLimit != 100000 || user.CreditsRemaining != 91201 || user.CreditsConsumed != 8799 {
		t.Errorf("credits = %d/%d/%d, do not match headers", user.CreditLimit, user.CreditsRemaining, user.CreditsConsumed)
	}
	if user.OptionsDataPermissions != "realtime" {
		t.Errorf("OptionsDataPermissions = %q, want realtime", user.OptionsDataPermissions)
	}
}
