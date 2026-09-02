// Example: Multi-Asset Dashboard
//
// Fan-out/fan-in dashboard that concurrently fetches stocks, options chains,
// fund candles, market status, API status, and account info, then merges
// results into a unified terminal view. Supports optional JSON output.
//
// Usage:
//
//	export MARKETDATA_TOKEN="your-api-key"
//	go run ./examples/multi-asset-dashboard
//	go run ./examples/multi-asset-dashboard -stocks AAPL,MSFT -options AAPL -fund SPY
//	go run ./examples/multi-asset-dashboard -json
package main

import (
	"context"
	"encoding/json"
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
	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/utilities"
)

// dashboard holds all fetched data from concurrent requests.
type dashboard struct {
	// Market overview
	MarketStatus *markets.MarketStatus `json:"marketStatus,omitempty"`
	APIStatus    *utilities.APIStatus  `json:"apiStatus,omitempty"`
	UserInfo     *utilities.UserInfo   `json:"userInfo,omitempty"`

	// Stock quotes
	Quotes []stocks.Quote `json:"quotes,omitempty"`

	// Options chain summary
	OptionsUnderlying string                `json:"optionsUnderlying,omitempty"`
	OptionsExpiration string                `json:"optionsExpiration,omitempty"`
	TopCalls          []options.OptionQuote `json:"topCalls,omitempty"`
	TopPuts           []options.OptionQuote `json:"topPuts,omitempty"`

	// Fund performance
	FundSymbol  string         `json:"fundSymbol,omitempty"`
	FundCandles []funds.Candle `json:"fundCandles,omitempty"`

	// Timing
	FetchDuration string `json:"fetchDuration"`
}

func main() {
	stocksFlag := flag.String("stocks", "AAPL,MSFT,GOOG,AMZN,TSLA,NVDA", "Comma-separated stock symbols")
	optionsFlag := flag.String("options", "AAPL", "Underlying for options chain snapshot")
	fundFlag := flag.String("fund", "VFIAX", "Mutual fund symbol for performance data (e.g., VFIAX, FXAIX)")
	jsonFlag := flag.Bool("json", false, "Output as JSON instead of formatted text")
	flag.Parse()

	stockSymbols := strings.Split(*stocksFlag, ",")
	for i := range stockSymbols {
		stockSymbols[i] = strings.TrimSpace(stockSymbols[i])
	}

	// Create SDK client
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()

	// Fan-out: launch all fetches concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	d := &dashboard{
		FundSymbol: *fundFlag,
	}

	// 1. Market status
	wg.Add(1)
	go func() {
		defer wg.Done()
		status, _, fetchErr := client.Markets.Status(ctx)
		if fetchErr != nil {
			log.Printf("Market status: %v", fetchErr)
			return
		}
		mu.Lock()
		d.MarketStatus = status
		mu.Unlock()
	}()

	// 2. API status
	wg.Add(1)
	go func() {
		defer wg.Done()
		status, _, fetchErr := client.Utilities.Status(ctx)
		if fetchErr != nil {
			log.Printf("API status: %v", fetchErr)
			return
		}
		mu.Lock()
		d.APIStatus = status
		mu.Unlock()
	}()

	// 3. User info
	wg.Add(1)
	go func() {
		defer wg.Done()
		user, _, fetchErr := client.Utilities.User(ctx)
		if fetchErr != nil {
			log.Printf("User info: %v", fetchErr)
			return
		}
		mu.Lock()
		d.UserInfo = user
		mu.Unlock()
	}()

	// 4. Stock quotes — one bulk request for the whole board. Bulk quotes
	// carry no 52-week data (single-quote only), so the renderer's 52-week
	// section simply stays hidden (see has52w below).
	wg.Add(1)
	go func() {
		defer wg.Done()
		quotes, _, fetchErr := client.Stocks.Quotes(ctx, stockSymbols)
		if fetchErr != nil {
			log.Printf("Stock quotes: %v", fetchErr)
			return
		}
		mu.Lock()
		d.Quotes = quotes
		mu.Unlock()
	}()

	// 5. Options chain (nearest expiration)
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Get nearest expiration
		exps, _, fetchErr := client.Options.Expirations(ctx, *optionsFlag)
		if fetchErr != nil || exps == nil || len(exps.Dates) == 0 {
			if fetchErr != nil {
				log.Printf("Options expirations: %v", fetchErr)
			}
			return
		}
		nearestExp := exps.Dates[0]

		// Fetch chain for nearest expiration, top by volume
		chain, _, fetchErr := client.Options.Chain(ctx, *optionsFlag,
			options.WithExpiry(options.OnExpiration(nearestExp)),
			options.WithMinVolume(10),
		)
		if fetchErr != nil || chain == nil {
			if fetchErr != nil {
				log.Printf("Options chain: %v", fetchErr)
			}
			return
		}

		// Split into calls and puts, take top 5 by volume
		var callsList, putsList []options.OptionQuote
		for _, opt := range chain.Options {
			if opt.Type == options.Call {
				callsList = append(callsList, opt)
			} else {
				putsList = append(putsList, opt)
			}
		}

		// Sort each by volume descending and take top 5. The sort is the
		// point: without it this returned the first five contracts the API
		// listed — strike order — under a "by volume" heading, hiding a
		// 1,367-volume contract behind four with 20 to 155.
		sortByVolume := func(s []options.OptionQuote) []options.OptionQuote {
			sorted := append([]options.OptionQuote(nil), s...)
			sort.SliceStable(sorted, func(i, j int) bool {
				return sorted[i].Volume > sorted[j].Volume
			})
			if len(sorted) > 5 {
				sorted = sorted[:5]
			}
			return sorted
		}

		mu.Lock()
		d.OptionsUnderlying = *optionsFlag
		d.OptionsExpiration = nearestExp.Format("2006-01-02")
		d.TopCalls = sortByVolume(callsList)
		d.TopPuts = sortByVolume(putsList)
		mu.Unlock()
	}()

	// 6. Fund candles (last 5 trading days)
	wg.Add(1)
	go func() {
		defer wg.Done()
		candles, _, fetchErr := client.Funds.Candles(ctx, *fundFlag,
			funds.WithResolution(funds.ResolutionDaily),
			funds.WithCandleWindow(funds.LastN(5)),
		)
		if fetchErr != nil {
			log.Printf("Fund candles: %v", fetchErr)
			return
		}
		mu.Lock()
		d.FundCandles = candles
		mu.Unlock()
	}()

	// Fan-in: wait for all to complete
	wg.Wait()
	d.FetchDuration = time.Since(start).Round(time.Millisecond).String()

	// Output
	if *jsonFlag {
		outputJSON(d)
	} else {
		outputText(d, client)
	}
}

func outputJSON(d *dashboard) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(d); err != nil {
		log.Fatalf("JSON encode error: %v", err)
	}
}

func outputText(d *dashboard, client *marketdata.Client) {
	fmt.Printf("Multi-Asset Dashboard (fetched in %s)\n", d.FetchDuration)
	fmt.Println(strings.Repeat("=", 70))

	// Market & API Status
	fmt.Println("\n--- STATUS ---")
	if d.MarketStatus != nil {
		fmt.Printf("Market: %s (%s)\n", d.MarketStatus.Status, d.MarketStatus.Date.Format("2006-01-02"))
	}
	if d.APIStatus != nil {
		fmt.Printf("API: %s (30d uptime: %.2f%%, 90d: %.2f%%)\n",
			d.APIStatus.Status, d.APIStatus.Uptime30d, d.APIStatus.Uptime90d)
	}
	if d.UserInfo != nil {
		fmt.Printf("Account: %d/%d credits remaining | Options: %s\n",
			d.UserInfo.CreditsRemaining, d.UserInfo.CreditLimit,
			d.UserInfo.OptionsDataPermissions)
	}

	// Stock Quotes
	if len(d.Quotes) > 0 {
		fmt.Println("\n--- STOCK QUOTES ---")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
		// Check if 52-week data is available
		has52w := false
		for _, q := range d.Quotes {
			if q.FiftyTwoWeekHigh > 0 {
				has52w = true
				break
			}
		}

		if has52w {
			fmt.Fprintln(w, "SYMBOL\tLAST\tCHANGE\tCHG%\tVOLUME\t52W HIGH\t52W LOW\t")
			fmt.Fprintln(w, "------\t------\t------\t------\t------\t--------\t-------\t")
		} else {
			fmt.Fprintln(w, "SYMBOL\tLAST\tCHANGE\tCHG%\tBID\tASK\tVOLUME\t")
			fmt.Fprintln(w, "------\t------\t------\t------\t---\t---\t------\t")
		}
		for _, q := range d.Quotes {
			chgPrefix := ""
			if q.Change > 0 {
				chgPrefix = "+"
			}
			if has52w {
				fmt.Fprintf(w, "%s\t$%.2f\t%s%.2f\t%s%.2f%%\t%d\t$%.2f\t$%.2f\t\n",
					q.Symbol, q.Last,
					chgPrefix, q.Change, chgPrefix, q.ChangePercent*100,
					q.Volume, q.FiftyTwoWeekHigh, q.FiftyTwoWeekLow)
			} else {
				fmt.Fprintf(w, "%s\t$%.2f\t%s%.2f\t%s%.2f%%\t$%.2f\t$%.2f\t%d\t\n",
					q.Symbol, q.Last,
					chgPrefix, q.Change, chgPrefix, q.ChangePercent*100,
					q.Bid, q.Ask, q.Volume)
			}
		}
		w.Flush()
	}

	// Options Chain
	if len(d.TopCalls) > 0 || len(d.TopPuts) > 0 {
		fmt.Printf("\n--- OPTIONS: %s (exp %s) ---\n", d.OptionsUnderlying, d.OptionsExpiration)

		if len(d.TopCalls) > 0 {
			fmt.Println("\nTop Calls by Volume:")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
			fmt.Fprintln(w, "STRIKE\tBID\tASK\tMID\tVOL\tOI\tIV\tDELTA\t")
			fmt.Fprintln(w, "------\t---\t---\t---\t---\t---\t---\t-----\t")
			for _, c := range d.TopCalls {
				fmt.Fprintf(w, "$%.2f\t$%.2f\t$%.2f\t$%.2f\t%d\t%d\t%.1f%%\t%.3f\t\n",
					c.Strike, c.Bid, c.Ask, c.CalcMid(), c.Volume, c.OpenInterest, c.IV*100, c.Delta)
			}
			w.Flush()
		}

		if len(d.TopPuts) > 0 {
			fmt.Println("\nTop Puts by Volume:")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
			fmt.Fprintln(w, "STRIKE\tBID\tASK\tMID\tVOL\tOI\tIV\tDELTA\t")
			fmt.Fprintln(w, "------\t---\t---\t---\t---\t---\t---\t-----\t")
			for _, p := range d.TopPuts {
				fmt.Fprintf(w, "$%.2f\t$%.2f\t$%.2f\t$%.2f\t%d\t%d\t%.1f%%\t%.3f\t\n",
					p.Strike, p.Bid, p.Ask, p.CalcMid(), p.Volume, p.OpenInterest, p.IV*100, p.Delta)
			}
			w.Flush()
		}
	}

	// Fund Performance
	if len(d.FundCandles) > 0 {
		fmt.Printf("\n--- FUND: %s (last %d days) ---\n", d.FundSymbol, len(d.FundCandles))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
		fmt.Fprintln(w, "DATE\tOPEN\tHIGH\tLOW\tCLOSE\tRANGE\t")
		fmt.Fprintln(w, "----\t----\t----\t---\t-----\t-----\t")
		for _, c := range d.FundCandles {
			fmt.Fprintf(w, "%s\t$%.2f\t$%.2f\t$%.2f\t$%.2f\t$%.2f\t\n",
				c.Time.Format("01/02"), c.Open, c.High, c.Low, c.Close, c.Range())
		}
		w.Flush()
	}

	// Footer
	// client.RateLimits() is a snapshot of the LAST completed response, not a
	// running total: Consumed is what that single request cost. A session
	// total means summing resp.RateLimit.Consumed across your own calls — see
	// the response-formats example.
	limits := client.RateLimits()
	fmt.Printf("\n--- Credits: %d remaining of %d (last request cost %d) ---\n",
		limits.Remaining, limits.Limit, limits.Consumed)
}
