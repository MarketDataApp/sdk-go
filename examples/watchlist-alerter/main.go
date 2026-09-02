// Example: Watchlist Alerter
//
// A channel-based alert system that monitors a watchlist of stocks and fires
// alerts when conditions are met (price crosses threshold, spread widens,
// volume spikes). Uses a 3-stage Go channel pipeline: fetch -> check -> alert.
//
// Usage:
//
//	export MARKETDATA_TOKEN="your-api-key"
//	go run ./examples/watchlist-alerter
//	go run ./examples/watchlist-alerter -symbols AAPL,TSLA -interval 15s -price-threshold 5.0
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// Alert represents a triggered alert condition.
type Alert struct {
	Symbol    string
	Condition string
	Message   string
	Time      time.Time
}

// snapshot holds the previous quote data for comparison.
type snapshot struct {
	last   float64
	volume int64
}

func main() {
	// Parse CLI flags
	symbolsFlag := flag.String("symbols", "AAPL,MSFT,GOOG,TSLA,NVDA", "Comma-separated list of stock symbols")
	intervalFlag := flag.Duration("interval", 30*time.Second, "Polling interval")
	priceThreshold := flag.Float64("price-threshold", 3.0, "Alert when price moves more than this % between polls")
	spreadThreshold := flag.Float64("spread-threshold", 1.0, "Alert when bid-ask spread exceeds this %")
	volumeThreshold := flag.Int64("volume-threshold", 0, "Alert when volume exceeds this level (0 = disabled)")
	flag.Parse()

	symbols := strings.Split(*symbolsFlag, ",")
	for i := range symbols {
		symbols[i] = strings.TrimSpace(symbols[i])
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create SDK client
	client, err := marketdata.NewClient()
	if err != nil {
		log(logger, "Failed to create client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("Watchlist Alerter")
	fmt.Printf("Tracking: %s\n", strings.Join(symbols, ", "))
	fmt.Printf("Interval: %s | Price threshold: %.1f%% | Spread threshold: %.1f%%\n",
		*intervalFlag, *priceThreshold, *spreadThreshold)
	if *volumeThreshold > 0 {
		fmt.Printf("Volume threshold: %d\n", *volumeThreshold)
	}
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()

	// Check market status first
	marketStatus, _, err := client.Markets.Status(ctx)
	if err != nil {
		log(logger, "Could not check market status", "error", err)
	} else if marketStatus != nil && marketStatus.IsClosed() {
		fmt.Printf("Note: Market is %s. Alerts may not trigger.\n\n", marketStatus.Status)
	}

	// --- 3-Stage Channel Pipeline ---

	// Stage 1 output: raw quotes from the fetcher
	quoteCh := make(chan []stocks.Quote, 1)

	// Stage 2 output: alerts from the condition checker
	alertCh := make(chan Alert, 10)

	// Stage 1: Fetcher goroutine — polls quotes on a ticker
	go func() {
		defer close(quoteCh)

		// Fetch immediately on startup
		if quotes := fetchQuotes(ctx, client, symbols, logger); quotes != nil {
			select {
			case quoteCh <- quotes:
			case <-ctx.Done():
				return
			}
		}

		ticker := time.NewTicker(*intervalFlag)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				quotes := fetchQuotes(ctx, client, symbols, logger)
				if quotes != nil {
					select {
					case quoteCh <- quotes:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	// Stage 2: Condition checker goroutine — compares quotes against thresholds
	go func() {
		defer close(alertCh)

		prev := make(map[string]snapshot)

		for quotes := range quoteCh {
			for _, q := range quotes {
				// Check price movement threshold
				if old, ok := prev[q.Symbol]; ok && old.last > 0 {
					pctMove := ((q.Last - old.last) / old.last) * 100
					if pctMove > *priceThreshold || pctMove < -*priceThreshold {
						direction := "UP"
						if pctMove < 0 {
							direction = "DOWN"
						}
						select {
						case alertCh <- Alert{
							Symbol:    q.Symbol,
							Condition: "PRICE_MOVE",
							Message:   fmt.Sprintf("%s moved %s %.2f%% ($%.2f -> $%.2f)", q.Symbol, direction, pctMove, old.last, q.Last),
							Time:      time.Now(),
						}:
						case <-ctx.Done():
							return
						}
					}
				}

				// Check spread threshold
				spreadPct := q.SpreadPercent()
				if spreadPct > *spreadThreshold {
					select {
					case alertCh <- Alert{
						Symbol:    q.Symbol,
						Condition: "WIDE_SPREAD",
						Message:   fmt.Sprintf("%s spread is %.2f%% (bid: $%.2f, ask: $%.2f)", q.Symbol, spreadPct, q.Bid, q.Ask),
						Time:      time.Now(),
					}:
					case <-ctx.Done():
						return
					}
				}

				// Check volume threshold
				if *volumeThreshold > 0 && q.Volume > *volumeThreshold {
					select {
					case alertCh <- Alert{
						Symbol:    q.Symbol,
						Condition: "HIGH_VOLUME",
						Message:   fmt.Sprintf("%s volume %d exceeds threshold %d", q.Symbol, q.Volume, *volumeThreshold),
						Time:      time.Now(),
					}:
					case <-ctx.Done():
						return
					}
				}

				// Check 52-week high/low proximity
				if q.FiftyTwoWeekHigh > 0 {
					pctFromHigh := ((q.FiftyTwoWeekHigh - q.Last) / q.FiftyTwoWeekHigh) * 100
					if pctFromHigh < 2.0 {
						select {
						case alertCh <- Alert{
							Symbol:    q.Symbol,
							Condition: "NEAR_52W_HIGH",
							Message:   fmt.Sprintf("%s is %.1f%% from 52-week high ($%.2f)", q.Symbol, pctFromHigh, q.FiftyTwoWeekHigh),
							Time:      time.Now(),
						}:
						case <-ctx.Done():
							return
						}
					}
				}

				if q.FiftyTwoWeekLow > 0 {
					pctFromLow := ((q.Last - q.FiftyTwoWeekLow) / q.FiftyTwoWeekLow) * 100
					if pctFromLow < 2.0 {
						select {
						case alertCh <- Alert{
							Symbol:    q.Symbol,
							Condition: "NEAR_52W_LOW",
							Message:   fmt.Sprintf("%s is %.1f%% from 52-week low ($%.2f)", q.Symbol, pctFromLow, q.FiftyTwoWeekLow),
							Time:      time.Now(),
						}:
						case <-ctx.Done():
							return
						}
					}
				}

				// Save snapshot for next comparison
				prev[q.Symbol] = snapshot{
					last:   q.Last,
					volume: q.Volume,
				}
			}
		}
	}()

	// Stage 3: Alert printer (main goroutine) — displays alerts
	alertCount := 0
	for alert := range alertCh {
		alertCount++
		fmt.Printf("[%s] %s: %s\n", alert.Time.Format("15:04:05"), alert.Condition, alert.Message)
	}

	fmt.Printf("\nShutting down. %d alerts fired.\n", alertCount)

	// Show credit usage
	// client.RateLimits() is a snapshot of the LAST completed response, not a
	// running total: Consumed is what that single request cost. A session
	// total means summing resp.RateLimit.Consumed across your own calls — see
	// the response-formats example.
	limits := client.RateLimits()
	fmt.Printf("Credits remaining: %d (last request cost %d)\n", limits.Remaining, limits.Consumed)
}

// fetchQuotes fetches quotes with 52-week data, handling rate limit errors
// gracefully. The 52-week alerts need per-symbol Quote calls: only the
// single-quote endpoint returns 52-week data (bulk quotes ignore it), at the
// cost of one credit per symbol per poll instead of one per poll.
func fetchQuotes(ctx context.Context, client *marketdata.Client, symbols []string, logger *slog.Logger) []stocks.Quote {
	quotes := make([]stocks.Quote, 0, len(symbols))
	for _, symbol := range symbols {
		q, _, err := client.Stocks.Quote(ctx, symbol, stocks.WithFiftyTwoWeek(true))
		if err != nil {
			if errors.Is(err, marketdata.ErrRateLimited) {
				log(logger, "Rate limited, skipping this poll cycle", "error", err)
			} else if ctx.Err() != nil {
				// Context cancelled, shutting down — not an error
				return nil
			} else {
				log(logger, "Error fetching quotes", "error", err)
			}
			return nil
		}
		if q != nil {
			quotes = append(quotes, *q)
		}
	}
	return quotes
}

// log is a helper to write structured log messages.
func log(logger *slog.Logger, msg string, args ...any) {
	logger.Info(msg, args...)
}
