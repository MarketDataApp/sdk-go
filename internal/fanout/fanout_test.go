package fanout

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestRun_ReturnsResultsInIndexOrder(t *testing.T) {
	// Finish in reverse order to prove the results are placed by index, not
	// by completion order.
	got, err := Run(context.Background(), 5, func(ctx context.Context, i int) (int, error) {
		time.Sleep(time.Duration(5-i) * time.Millisecond)
		return i * 10, nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for i, v := range got {
		if v != i*10 {
			t.Errorf("got[%d] = %d, want %d", i, v, i*10)
		}
	}
}

func TestRun_RunsConcurrently(t *testing.T) {
	// Each fetch blocks until every fetch has started; a sequential
	// implementation would deadlock and time out.
	const n = 4
	started := make(chan struct{}, n)
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := Run(context.Background(), n, func(ctx context.Context, i int) (int, error) {
			started <- struct{}{}
			<-release
			return i, nil
		})
		done <- err
	}()
	for i := 0; i < n; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d of %d fetches started; Run is not concurrent", i, n)
		}
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRun_FirstFailureCancelsSiblings(t *testing.T) {
	failure := errors.New("boom")
	var cancelled atomic.Int32
	_, err := Run(context.Background(), 8, func(ctx context.Context, i int) (int, error) {
		if i == 0 {
			return 0, failure
		}
		select {
		case <-ctx.Done():
			cancelled.Add(1)
			return 0, ctx.Err()
		case <-time.After(2 * time.Second):
			return i, nil
		}
	})
	if !errors.Is(err, failure) {
		t.Fatalf("Run() error = %v, want the root failure", err)
	}
	if n := cancelled.Load(); n != 7 {
		t.Errorf("%d siblings observed cancellation, want 7", n)
	}
}

func TestRun_PrefersRootErrorOverCancellationEchoes(t *testing.T) {
	// The failing fetch is deliberately the slowest, so several
	// context.Canceled echoes land in the results before it does. Run must
	// still report the cause, not an echo.
	failure := errors.New("unauthorized")
	_, err := Run(context.Background(), 10, func(ctx context.Context, i int) (int, error) {
		if i == 9 {
			return 0, failure
		}
		<-ctx.Done()
		return 0, ctx.Err()
	})
	if !errors.Is(err, failure) {
		t.Fatalf("Run() error = %v, want the root failure, not a cancellation echo", err)
	}
}

func TestRun_AlreadyCancelledContextReportsCancellation(t *testing.T) {
	// When the caller's own context is already dead every fetch fails that
	// way, so there is no non-cancellation error to prefer and reporting
	// the cancellation is correct.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Run(ctx, 3, func(ctx context.Context, i int) (int, error) {
		return 0, ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
}

func TestRun_IsAllOrNothing(t *testing.T) {
	// A single failure discards every successful sibling's result rather
	// than returning a partially-populated slice.
	got, err := Run(context.Background(), 3, func(ctx context.Context, i int) (int, error) {
		if i == 1 {
			return 0, errors.New("boom")
		}
		return i, nil
	})
	if err == nil {
		t.Fatal("Run() should return the failure")
	}
	if got != nil {
		t.Errorf("Run() results = %v, want nil alongside an error", got)
	}
}

func TestRun_ZeroItems(t *testing.T) {
	got, err := Run(context.Background(), 0, func(ctx context.Context, i int) (int, error) {
		t.Error("fetch should not be called for n = 0")
		return 0, nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Run() results = %v, want empty", got)
	}
}

func TestRun_PropagatesFetchIndex(t *testing.T) {
	// Each fetch must see its own index; a closure capturing the loop
	// variable incorrectly would collapse them.
	got, err := Run(context.Background(), 4, func(ctx context.Context, i int) (string, error) {
		return fmt.Sprintf("item-%d", i), nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for i, v := range got {
		if want := fmt.Sprintf("item-%d", i); v != want {
			t.Errorf("got[%d] = %q, want %q", i, v, want)
		}
	}
}

func TestIsChild(t *testing.T) {
	if IsChild(context.Background()) {
		t.Error("IsChild(Background) = true, want false")
	}

	var inside context.Context
	if _, err := Run(context.Background(), 1, func(ctx context.Context, _ int) (int, error) {
		inside = ctx
		return 0, nil
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !IsChild(inside) {
		t.Error("IsChild(ctx from Run) = false, want true")
	}

	// The mark survives further derivation, which is what makes it usable
	// deep in the request path.
	derived, cancel := context.WithCancel(inside)
	defer cancel()
	if !IsChild(derived) {
		t.Error("IsChild should survive a derived context")
	}
}
