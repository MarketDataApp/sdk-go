// Example: Covered Call Screener
//
// Scans options chains for multiple underlyings concurrently, finding covered
// call candidates filtered by DTE, volume, spread, and moneyness. Uses
// goroutine-per-symbol fan-out and demonstrates typed error handling.
//
// Usage:
//
//	export MARKETDATA_TOKEN="your-api-key"
//	go run ./examples/covered-call-screener
//	go run ./examples/covered-call-screener -symbols AAPL,MSFT,GOOG -dte 30 -min-volume 100 -side call
//	go run ./examples/covered-call-screener -symbols AAPL -side put -top 10
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

// candidate represents a screened option contract with its underlying price.
type candidate struct {
	options.OptionQuote
	UnderlyingLast float64
	Premium        float64 // Annualized premium return
}

func main() {
	symbolsFlag := flag.String("symbols", "AAPL,MSFT,GOOG,AMZN,META", "Comma-separated underlyings to scan")
	sideFlag := flag.String("side", "call", "Option side: call or put")
	dteFlag := flag.Int("dte", 45, "Target days to expiration")
	minVolumeFlag := flag.Int("min-volume", 50, "Minimum volume filter")
	minOIFlag := flag.Int("min-oi", 100, "Minimum open interest filter")
	maxSpreadFlag := flag.Float64("max-spread-pct", 15.0, "Maximum bid-ask spread percentage")
	topFlag := flag.Int("top", 20, "Number of top results to display")
	flag.Parse()

	symbols := strings.Split(*symbolsFlag, ",")
	for i := range symbols {
		symbols[i] = strings.TrimSpace(symbols[i])
	}

	side := options.SideCall
	if *sideFlag == "put" {
		side = options.SidePut
	}

	// Create SDK client
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	strategy := "Covered Call"
	if side == options.SidePut {
		strategy = "Cash-Secured Put"
	}
	fmt.Printf("%s Screener — scanning %d underlyings\n", strategy, len(symbols))
	fmt.Printf("Filters: DTE ~%d, min volume %d, min OI %d, max spread %.1f%%\n\n",
		*dteFlag, *minVolumeFlag, *minOIFlag, *maxSpreadFlag)

	start := time.Now()

	// Fan-out: scan each symbol concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allCandidates []candidate
	var scanErrors []string

	for _, sym := range symbols {
		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()

			candidates, scanErr := scanSymbol(ctx, client, symbol, side,
				*dteFlag, *minVolumeFlag, *minOIFlag, *maxSpreadFlag)

			mu.Lock()
			defer mu.Unlock()

			if scanErr != nil {
				scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", symbol, scanErr))
				return
			}

			allCandidates = append(allCandidates, candidates...)
			fmt.Printf("  [%s] found %d candidates\n", symbol, len(candidates))
		}(sym)
	}
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Printf("\nScan complete in %s\n", elapsed.Round(time.Millisecond))

	// Report errors
	if len(scanErrors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(scanErrors))
		for _, e := range scanErrors {
			fmt.Printf("  %s\n", e)
		}
	}

	if len(allCandidates) == 0 {
		fmt.Println("\nNo candidates found matching the criteria.")
		return
	}

	// Sort by annualized premium return (descending)
	sort.Slice(allCandidates, func(i, j int) bool {
		return allCandidates[i].Premium > allCandidates[j].Premium
	})

	// Display top results
	limit := *topFlag
	if limit > len(allCandidates) {
		limit = len(allCandidates)
	}

	fmt.Printf("\nTop %d %s candidates (of %d total):\n\n", limit, strategy, len(allCandidates))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "SYMBOL\tSTRIKE\tEXP\tDTE\tBID\tASK\tMID\tSPREAD\tVOL\tOI\tIV\tDELTA\tANN. PREM%\t")
	fmt.Fprintln(w, "------\t------\t------\t---\t---\t---\t---\t------\t---\t---\t---\t-----\t---------\t")

	for _, c := range allCandidates[:limit] {
		fmt.Fprintf(w, "%s\t$%.2f\t%s\t%d\t$%.2f\t$%.2f\t$%.2f\t$%.2f\t%d\t%d\t%.1f%%\t%.3f\t%.1f%%\t\n",
			c.Underlying,
			c.Strike,
			c.Expiration.Format("01/02"),
			c.DTE,
			c.Bid,
			c.Ask,
			c.CalcMid(),
			c.Spread(),
			c.Volume,
			c.OpenInterest,
			c.IV*100,
			c.Delta,
			c.Premium,
		)
	}
	w.Flush()

	// Credit usage
	// client.RateLimits() is a snapshot of the LAST completed response, not a
	// running total: Consumed is what that single request cost. A session
	// total means summing resp.RateLimit.Consumed across your own calls — see
	// the response-formats example.
	limits := client.RateLimits()
	fmt.Printf("\nCredits remaining: %d (last request cost %d)\n", limits.Remaining, limits.Consumed)
}

// scanSymbol scans a single underlying for option candidates.
func scanSymbol(ctx context.Context, client *marketdata.Client, symbol string, side options.OptionSide,
	targetDTE, minVolume, minOI int, maxSpreadPct float64) ([]candidate, error) {

	// First get the nearest expiration to target DTE
	expirations, _, err := client.Options.Expirations(ctx, symbol)
	if err != nil {
		return nil, handleError(err)
	}
	if expirations == nil || len(expirations.Dates) == 0 {
		return nil, fmt.Errorf("no expirations available")
	}

	// Find the expiration closest to target DTE
	targetDate := time.Now().AddDate(0, 0, targetDTE)
	bestExp := expirations.Dates[0]
	bestDiff := abs(int(targetDate.Sub(bestExp).Hours() / 24))
	for _, exp := range expirations.Dates[1:] {
		diff := abs(int(targetDate.Sub(exp).Hours() / 24))
		if diff < bestDiff {
			bestDiff = diff
			bestExp = exp
		}
	}

	// Fetch the options chain with server-side filters
	chainOpts := []options.ChainOption{
		options.WithExpiry(options.OnExpiration(bestExp)),
		options.WithSide(side),
		options.WithRange(options.MoneynessOTM),
		options.WithMinVolume(minVolume),
		options.WithMinOpenInterest(minOI),
		options.WithMaxBidAskSpreadPct(maxSpreadPct),
	}

	chain, _, err := client.Options.Chain(ctx, symbol, chainOpts...)
	if err != nil {
		return nil, handleError(err)
	}
	if chain == nil || len(chain.Options) == 0 {
		return nil, nil
	}

	// Build candidates with annualized premium calculation
	var candidates []candidate
	for _, opt := range chain.Options {
		if opt.Bid <= 0 || opt.DTE <= 0 {
			continue
		}

		// Annualized premium return = (mid / underlying price) * (365 / DTE) * 100
		mid := opt.CalcMid()
		underlying := opt.UnderlyingPrice
		if underlying <= 0 {
			continue
		}

		annualizedPremium := (mid / underlying) * (365.0 / float64(opt.DTE)) * 100

		candidates = append(candidates, candidate{
			OptionQuote:    opt,
			UnderlyingLast: underlying,
			Premium:        annualizedPremium,
		})
	}

	return candidates, nil
}

// handleError inspects SDK errors and returns a user-friendly message.
func handleError(err error) error {
	var rateLimitErr *marketdata.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return fmt.Errorf("rate limited (resets %s)", rateLimitErr.ResetAt.Format("15:04:05"))
	}
	if errors.Is(err, marketdata.ErrAuthentication) {
		return fmt.Errorf("authentication failed — check MARKETDATA_TOKEN")
	}
	// Note: there is no ErrNotFound branch on purpose — the API reports an
	// unknown symbol as "no data" (nil result, nil error), never as an error.
	return err
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
