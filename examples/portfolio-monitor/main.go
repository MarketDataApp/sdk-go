// Example: Portfolio Monitor
//
// A terminal-based portfolio monitor that polls stock prices on a configurable
// interval with graceful shutdown on Ctrl+C. Demonstrates Go's signal handling,
// context cancellation, time.Ticker, and select statement.
//
// Usage:
//
//	export MARKETDATA_TOKEN="your-api-key"
//	go run ./examples/portfolio-monitor
//	go run ./examples/portfolio-monitor -symbols AAPL,MSFT,GOOG,AMZN -interval 30s
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func main() {
	// Parse CLI flags
	symbolsFlag := flag.String("symbols", "AAPL,MSFT,GOOG,AMZN,TSLA,NVDA", "Comma-separated list of stock symbols")
	intervalFlag := flag.Duration("interval", 30*time.Second, "Polling interval (e.g., 10s, 1m)")
	flag.Parse()

	symbols := strings.Split(*symbolsFlag, ",")
	for i := range symbols {
		symbols[i] = strings.TrimSpace(symbols[i])
	}

	// Create SDK client
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Set up graceful shutdown: Ctrl+C or SIGTERM cancels the context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Portfolio Monitor — tracking %d symbols every %s\n", len(symbols), *intervalFlag)
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

	// Check market status
	marketStatus, _, err := client.Markets.Status(ctx)
	if err != nil {
		log.Printf("Warning: could not check market status: %v", err)
	} else if note := marketStatusNote(marketStatus); note != "" {
		fmt.Print(note)
	}

	// Initial fetch with full quotes (includes spread, volume, 52-week data)
	fetchAndDisplayQuotes(ctx, client, symbols)

	// Polling loop using time.Ticker
	ticker := time.NewTicker(*intervalFlag)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down...")
			displayCredits(client)
			return
		case <-ticker.C:
			// Use Prices for lightweight polling (SmartMid midpoint)
			fetchAndDisplayPrices(ctx, client, symbols)
		}
	}
}

// fetchAndDisplayQuotes fetches full quotes and renders a detailed table.
// 52-week data only exists on the single-quote endpoint (bulk quotes ignore
// it), so the full view fetches each symbol individually; the lightweight
// polling path below sticks to the one-request Prices call.
func fetchAndDisplayQuotes(ctx context.Context, client *marketdata.Client, symbols []string) {
	quotes := make([]stocks.Quote, 0, len(symbols))
	var resp *marketdata.Response
	for _, symbol := range symbols {
		q, r, err := client.Stocks.Quote(ctx, symbol, stocks.WithFiftyTwoWeek(true))
		if err != nil {
			log.Printf("Error fetching quote for %s: %v", symbol, err)
			return
		}
		if q != nil {
			quotes = append(quotes, *q)
		}
		resp = r
	}

	// Clear screen (ANSI escape)
	fmt.Print("\033[2J\033[H")
	fmt.Printf("Portfolio Monitor — %s (full quote)\n\n", time.Now().Format("15:04:05"))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "SYMBOL\tLAST\tCHANGE\tCHG%\tBID\tASK\tSPREAD\tVOLUME\t52W HIGH\t52W LOW\t")
	fmt.Fprintln(w, "------\t------\t------\t------\t------\t------\t------\t------\t--------\t-------\t")

	for _, q := range quotes {
		chgPrefix := ""
		if q.Change > 0 {
			chgPrefix = "+"
		}
		fmt.Fprintf(w, "%s\t$%.2f\t%s%.2f\t%s%.2f%%\t$%.2f\t$%.2f\t$%.2f\t%d\t$%.2f\t$%.2f\t\n",
			q.Symbol,
			q.Last,
			chgPrefix, q.Change,
			chgPrefix, q.ChangePercent*100,
			q.Bid,
			q.Ask,
			q.Spread(),
			q.Volume,
			q.FiftyTwoWeekHigh,
			q.FiftyTwoWeekLow,
		)
	}
	w.Flush()

	if resp != nil {
		fmt.Printf("\nCredits: %d remaining\n", resp.RateLimit.Remaining)
	}
}

// fetchAndDisplayPrices fetches lightweight SmartMid prices and renders a compact table.
func fetchAndDisplayPrices(ctx context.Context, client *marketdata.Client, symbols []string) {
	prices, resp, err := client.Stocks.Prices(ctx, symbols)
	if err != nil {
		log.Printf("Error fetching prices: %v", err)
		return
	}

	// Clear screen
	fmt.Print("\033[2J\033[H")
	fmt.Printf("Portfolio Monitor — %s (price update)\n\n", time.Now().Format("15:04:05"))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "SYMBOL\tMID PRICE\tCHANGE\tCHG%\tUPDATED\t")
	fmt.Fprintln(w, "------\t---------\t------\t------\t-------\t")

	for _, p := range prices {
		chgPrefix := ""
		if p.Change > 0 {
			chgPrefix = "+"
		}
		fmt.Fprintf(w, "%s\t$%.2f\t%s%.2f\t%s%.2f%%\t%s\t\n",
			p.Symbol,
			p.Mid,
			chgPrefix, p.Change,
			chgPrefix, p.ChangePercent*100,
			p.Updated.Format("15:04:05"),
		)
	}
	w.Flush()

	if resp != nil {
		fmt.Printf("\nCredits: %d remaining\n", resp.RateLimit.Remaining)
	}

	// Warn if credits are getting low
	limits := client.RateLimits()
	if limits.Remaining > 0 && limits.Remaining < 100 {
		fmt.Printf("WARNING: Only %d credits remaining!\n", limits.Remaining)
	}
}

// displayCredits shows the credit position on shutdown.
//
// It deliberately does not claim a session total. client.RateLimits() is a
// snapshot of the LAST completed response: Consumed is what that single
// request cost, not a running sum. Reporting it as "session credits used" —
// as this example previously did — misreads the API. A real session total
// means summing resp.RateLimit.Consumed across your own calls; see the
// response-formats example.
func displayCredits(client *marketdata.Client) {
	limits := client.RateLimits()
	fmt.Printf("Credits remaining: %d (last request cost %d)\n", limits.Remaining, limits.Consumed)
}

// marketStatusNote returns the message to print about the market's current
// status, or an empty string when there is nothing to report. status is nil
// when Markets.Status returned NoData (a valid, non-error outcome) — that
// case, and only that case, must not be dereferenced.
func marketStatusNote(status *markets.MarketStatus) string {
	switch {
	case status == nil:
		return ""
	case status.IsClosed():
		return fmt.Sprintf("Note: Market is currently %s (%s). Prices may be stale.\n\n",
			status.Status, status.Date.Format("2006-01-02"))
	default:
		return fmt.Sprintf("Market is %s.\n\n", status.Status)
	}
}
