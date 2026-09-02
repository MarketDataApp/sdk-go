package options

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// notFoundServer answers every request with the API's real "symbol not
// found" body: a 404 that names an error rather than reporting an empty
// answer. Captured live on 2026-09-01 from options/chain/ZZZZQQ/.
func notFoundHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"s":"no_data","errmsg":"Symbol not found."}`))
	})
}

// emptyAnswerHandler answers with the API's real markerless no-data body:
// a valid query whose answer is the empty set. Captured live the same day
// from options/chain/AAPL/ with an unsatisfiable minOpenInterest.
func emptyAnswerHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"s":"no_data","nextTime":null,"prevTime":null}`))
	})
}

// A symbol that does not exist is a caller error the API discriminates, so
// it must fail loudly. Before this, Chain answered it exactly as it answers
// an empty filter — a nil chain and a nil error — so a typo read as "this
// underlying has no contracts".
func TestChain_NonexistentSymbolIsNotFoundError(t *testing.T) {
	svc := newTestService(notFoundHandler())

	chain, _, err := svc.Chain(context.Background(), "ZZZZQQ")
	if err == nil {
		t.Fatalf("Chain() error = nil, want NotFoundError (got chain %+v)", chain)
	}
	var nf *sdkerrors.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("Chain() error = %T (%v), want *sdkerrors.NotFoundError", err, err)
	}
	if nf.Message != "Symbol not found." {
		t.Errorf("Message = %q, want the API's own errmsg", nf.Message)
	}
}

// The counterpart contract: a valid query matching nothing is not an error,
// and its result must be safe to range. This is the shape that crashed the
// integration binary in the PR #33 CI run (TestReflect_OptionsFilters/
// MinOpenInterest) — the SDK's own authors dereferenced the nil.
func TestChain_EmptyAnswerIsRangeableNotNil(t *testing.T) {
	svc := newTestService(emptyAnswerHandler())

	chain, resp, err := svc.Chain(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Chain() error = %v, want nil for an empty answer", err)
	}
	if chain == nil {
		t.Fatal("Chain() returned a nil chain; ranging it panics, which is the defect")
	}
	count := 0
	for range chain.Options {
		count++
	}
	if count != 0 {
		t.Errorf("ranged %d options, want 0", count)
	}
	if resp == nil || !resp.NoData {
		t.Errorf("response = %v, want NoData true", resp)
	}
}

func TestExpirations_NonexistentSymbolIsNotFoundError(t *testing.T) {
	svc := newTestService(notFoundHandler())

	if _, _, err := svc.Expirations(context.Background(), "ZZZZQQ"); !errors.Is(err, sdkerrors.ErrNotFound) {
		t.Fatalf("Expirations() error = %v, want ErrNotFound", err)
	}
}

func TestExpirations_EmptyAnswerIsRangeableNotNil(t *testing.T) {
	svc := newTestService(emptyAnswerHandler())

	exps, _, err := svc.Expirations(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Expirations() error = %v, want nil", err)
	}
	if exps == nil {
		t.Fatal("Expirations() returned nil; ranging Dates on it panics")
	}
	for range exps.Dates {
		t.Error("want no dates")
	}
}

// Quote is scalar-shaped, so the empty answer stays nil on purpose: every
// field of an OptionQuote is a price, and a zero-valued struct would read
// as a real quote of zero — a silently wrong answer, worse than the nil.
// Only the error case changes.
func TestQuote_NonexistentContractIsNotFoundError(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		// The real body for a well-formed OCC symbol matching no contract
		// (verified live 2026-09-01): note s is "error", not "no_data".
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"No option found. No option was found for this strike and expiration."}`))
	}))

	quote, _, err := svc.Quote(context.Background(), "ZZZZQQ260101C00100000")
	if err == nil {
		t.Fatalf("Quote() error = nil, want NotFoundError (got quote %+v)", quote)
	}
	if !errors.Is(err, sdkerrors.ErrNotFound) {
		t.Fatalf("Quote() error = %v, want ErrNotFound", err)
	}
}

func TestQuote_EmptyAnswerStaysNil(t *testing.T) {
	svc := newTestService(emptyAnswerHandler())

	quote, resp, err := svc.Quote(context.Background(), "AAPL260821C00300000")
	if err != nil {
		t.Fatalf("Quote() error = %v, want nil", err)
	}
	if quote != nil {
		t.Errorf("Quote() = %+v, want nil: a zero-valued OptionQuote is a wrong answer, not an empty one", quote)
	}
	if resp == nil || !resp.NoData {
		t.Errorf("response = %v, want NoData true", resp)
	}
}

// In a batch, an unknown contract is information about that one symbol, not
// a failure of the request. Letting it error would cancel the siblings
// (fanout stops on the first hard error), throwing away the credits already
// spent and breaking the promise QuotesBySymbol exists to keep: one entry
// per requested symbol.
func TestQuotesBySymbol_UnknownContractIsANilEntryNotABatchFailure(t *testing.T) {
	const real = "AAPL260821C00300000"
	const bogus = "AAPL260821C99999000"

	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, bogus) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"s":"error","errmsg":"No option found. No option was found for this strike and expiration."}`))
			return
		}
		_, _ = w.Write([]byte(`{"s":"ok","optionSymbol":["` + real + `"],"underlying":["AAPL"],"strike":[300],"side":["call"],"bid":[1.0],"ask":[1.1],"last":[1.05],"expiration":[1787000000],"updated":[1787000000]}`))
	}))

	quotes, resp, err := svc.QuotesBySymbol(context.Background(), []string{real, bogus})
	if err != nil {
		t.Fatalf("QuotesBySymbol() error = %v; one unknown contract must not fail the batch", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("len(quotes) = %d, want 2 (one entry per requested symbol)", len(quotes))
	}
	if q, ok := quotes[real]; !ok || q == nil {
		t.Errorf("quotes[%q] = %v (present=%v), want a quote", real, q, ok)
	}
	if q, ok := quotes[bogus]; !ok {
		t.Errorf("quotes[%q] missing: an unknown contract must still have an entry", bogus)
	} else if q != nil {
		t.Errorf("quotes[%q] = %v, want nil", bogus, q)
	}
	if resp == nil {
		t.Error("response = nil; the batch must still report request metadata")
	}

	// Quotes, by contrast, drops it entirely — same tolerance, different shape.
	slice, _, err := svc.Quotes(context.Background(), []string{real, bogus})
	if err != nil {
		t.Fatalf("Quotes() error = %v; one unknown contract must not fail the batch", err)
	}
	if len(slice) != 1 {
		t.Errorf("len(Quotes()) = %d, want 1 — the unknown symbol is omitted", len(slice))
	}
}

// A batch of exactly one takes a different code path (no fan-out), so it is
// pinned separately: it must agree with the batch, not with Quote.
func TestQuotesBySymbol_SingleUnknownContractIsANilEntry(t *testing.T) {
	const bogus = "AAPL260821C99999000"
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"No option found."}`))
	}))

	quotes, resp, err := svc.QuotesBySymbol(context.Background(), []string{bogus})
	if err != nil {
		t.Fatalf("QuotesBySymbol() error = %v, want nil for a one-symbol batch", err)
	}
	if q, ok := quotes[bogus]; !ok || q != nil {
		t.Errorf("quotes[%q] = %v (present=%v), want a present nil entry", bogus, q, ok)
	}
	if resp == nil || !resp.NoData {
		t.Errorf("response = %v, want a non-nil NoData response", resp)
	}
}

// A real error inside a batch still fails the batch: the tolerance is scoped
// to not-found, not to every failure.
func TestQuotesBySymbol_RealErrorStillFailsTheBatch(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"Invalid token"}`))
	}))

	if _, _, err := svc.QuotesBySymbol(context.Background(), []string{"A260821C00300000", "B260821C00300000"}); err == nil {
		t.Fatal("QuotesBySymbol() error = nil, want the 401 to fail the batch")
	}
}
