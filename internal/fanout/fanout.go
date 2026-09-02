// Package fanout carries the SDK's one concurrent-fetch policy: run N
// requests at once, abandon the rest as soon as one fails, and report the
// failure that actually caused the abandonment (ADR-014).
//
// It exists because that policy was written out four times — the options
// per-symbol quote batch, the stocks candle-range split, and both formatted
// fetch helpers in internal/response — with only the per-item fetch and the
// merge of successes differing between them. Those two are the parameters
// here. Keeping the policy in one place means a change to it (demoting a
// log level, preferring a different root error) lands once instead of four
// times, with nothing to detect a missed site.
package fanout

import (
	"context"
	"errors"
	"sync"
)

// Run calls fetch once for each index in [0, n) concurrently and returns
// the results in index order.
//
// The first failure cancels the context passed to every other fetch, so the
// siblings stop rather than keep spending API credits on a batch that is
// already going to fail. Because of that, Run is all-or-nothing: it returns
// either n results or one error, never a partial slice. Callers that want
// per-item outcomes (a map keyed by symbol, say) build them from the full
// slice after Run returns.
//
// The error Run reports is the one that caused the cancellation, not one of
// its echoes: every abandoned sibling fails with context.Canceled, and
// whichever of them lands in the results first would otherwise mask the
// real cause. Run therefore prefers any non-cancellation error over a
// cancellation. If the caller's own context was already cancelled, every
// fetch fails that way and Run reports it, which is correct.
//
// Run does not bound concurrency itself — the HTTP client's shared pool
// already does (ADR-014), and duplicating the limit here would deadlock a
// caller that fans out from inside another fan-out.
func Run[T any](ctx context.Context, n int, fetch func(ctx context.Context, i int) (T, error)) ([]T, error) {
	type result struct {
		value T
		err   error
	}
	results := make([]result, n)

	ctx, cancel := context.WithCancel(markChild(ctx))
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			value, err := fetch(ctx, idx)
			results[idx] = result{value: value, err: err}
			if err != nil {
				cancel()
			}
		}(i)
	}
	wg.Wait()

	var rootErr error
	for _, r := range results {
		if r.err == nil {
			continue
		}
		if rootErr == nil || (errors.Is(rootErr, context.Canceled) && !errors.Is(r.err, context.Canceled)) {
			rootErr = r.err
		}
	}
	if rootErr != nil {
		return nil, rootErr
	}

	values := make([]T, n)
	for i, r := range results {
		values[i] = r.value
	}
	return values, nil
}

// childKey marks a context as belonging to a request Run issued.
type childKey struct{}

func markChild(ctx context.Context) context.Context {
	return context.WithValue(ctx, childKey{}, true)
}

// IsChild reports whether ctx belongs to one of the concurrent requests a
// Run is driving.
//
// It exists so logging can tell one logical failure from its N echoes. When
// the caller's own context expires, every sibling fails with
// DeadlineExceeded, and a log point that sees only the error cannot tell
// that from N independent timeouts — so a 50-symbol batch under a
// context.WithTimeout produced 50 ERROR lines for one expiry. Run reports a
// single error to its caller, so a context failure on a child it drove is
// never news on its own.
//
// Only context failures should be demoted on this basis. A 401 or a 500 on
// one sibling is a real, distinct server answer and still deserves its line.
func IsChild(ctx context.Context) bool {
	marked, _ := ctx.Value(childKey{}).(bool)
	return marked
}
