// Example: Historical Data Exporter
//
// Bulk CSV exporter for historical candle data. Fetches candles for multiple
// symbols concurrently and writes each to a CSV file. For intraday resolutions
// with large date ranges, the SDK automatically splits requests into year-sized
// chunks and fetches them in parallel.
//
// Usage:
//
//	export MARKETDATA_TOKEN="your-api-key"
//	go run ./examples/historical-exporter
//	go run ./examples/historical-exporter -symbols AAPL,MSFT -resolution D -from 2024-01-01 -to 2025-01-01
//	go run ./examples/historical-exporter -symbols SPY -type fund -resolution M -from 2020-01-01
//	go run ./examples/historical-exporter -symbols AAPL -resolution 5 -from 2024-06-01 -to 2025-06-01 -outdir ./data
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// candle is a unified candle representation for CSV output.
type candle struct {
	Symbol string
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

func main() {
	symbolsFlag := flag.String("symbols", "AAPL,MSFT,GOOG", "Comma-separated list of symbols")
	resolutionFlag := flag.String("resolution", "D", "Resolution: 1,3,5,15,30,45,60,120,240,D,W,M (stocks) or D,W,M,Q,Y (funds)")
	fromFlag := flag.String("from", "", "Start date (YYYY-MM-DD). Defaults to 1 year ago.")
	toFlag := flag.String("to", "", "End date (YYYY-MM-DD). Defaults to today.")
	// Defaults under the OS temp dir, not the working directory: running
	// this example from the repository root dropped three untracked CSVs
	// into it, and .gitignore did not cover them.
	outdirFlag := flag.String("outdir", filepath.Join(os.TempDir(), "marketdata-export"), "Output directory for CSV files")
	typeFlag := flag.String("type", "stock", "Asset type: stock or fund")
	flag.Parse()

	symbols := strings.Split(*symbolsFlag, ",")
	for i := range symbols {
		symbols[i] = strings.TrimSpace(symbols[i])
	}

	// Parse dates
	now := time.Now()
	from := now.AddDate(-1, 0, 0) // Default: 1 year ago
	to := now

	if *fromFlag != "" {
		parsed, err := time.Parse("2006-01-02", *fromFlag)
		if err != nil {
			log.Fatalf("Invalid -from date: %v", err)
		}
		from = parsed
	}
	if *toFlag != "" {
		parsed, err := time.Parse("2006-01-02", *toFlag)
		if err != nil {
			log.Fatalf("Invalid -to date: %v", err)
		}
		to = parsed
	}

	// Create output directory if needed
	if err := os.MkdirAll(*outdirFlag, 0755); err != nil {
		log.Fatalf("Cannot create output directory: %v", err)
	}

	// Create SDK client
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Printf("Historical Exporter — %d symbols, %s resolution, %s to %s\n",
		len(symbols), *resolutionFlag, from.Format("2006-01-02"), to.Format("2006-01-02"))
	fmt.Printf("Output directory: %s\n\n", *outdirFlag)

	// Fetch all symbols concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalCandles int
	var errorCount int

	start := time.Now()

	for _, sym := range symbols {
		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()

			var candles []candle
			var fetchErr error

			switch *typeFlag {
			case "fund":
				candles, fetchErr = fetchFundCandles(ctx, client, symbol, *resolutionFlag, from, to)
			default:
				candles, fetchErr = fetchStockCandles(ctx, client, symbol, *resolutionFlag, from, to)
			}

			if fetchErr != nil {
				log.Printf("[%s] Error: %v", symbol, fetchErr)
				mu.Lock()
				errorCount++
				mu.Unlock()
				return
			}

			if len(candles) == 0 {
				log.Printf("[%s] No data returned", symbol)
				return
			}

			// Write CSV
			filename := fmt.Sprintf("%s_%s_%s_%s.csv",
				symbol, *resolutionFlag, from.Format("20060102"), to.Format("20060102"))
			path := filepath.Join(*outdirFlag, filename)

			if writeErr := writeCSV(path, candles); writeErr != nil {
				log.Printf("[%s] CSV write error: %v", symbol, writeErr)
				mu.Lock()
				errorCount++
				mu.Unlock()
				return
			}

			// Print summary stats
			first := candles[0]
			last := candles[len(candles)-1]
			pctChange := 0.0
			if first.Open > 0 {
				pctChange = ((last.Close - first.Open) / first.Open) * 100
			}

			mu.Lock()
			totalCandles += len(candles)
			mu.Unlock()

			fmt.Printf("[%s] %d candles -> %s (%.2f%% over period)\n",
				symbol, len(candles), path, pctChange)
		}(sym)
	}
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Printf("\nDone: %d candles exported in %s (%d errors)\n", totalCandles, elapsed.Round(time.Millisecond), errorCount)

	// client.RateLimits() is a snapshot of the LAST completed response, not a
	// running total: Consumed is what that single request cost. A session
	// total means summing resp.RateLimit.Consumed across your own calls — see
	// the response-formats example.
	limits := client.RateLimits()
	fmt.Printf("Credits remaining: %d (last request cost %d)\n", limits.Remaining, limits.Consumed)
}

// fetchStockCandles fetches stock candles using the SDK.
// For intraday resolutions with ranges > 1 year, the SDK automatically
// splits into year-sized chunks and fetches concurrently.
func fetchStockCandles(ctx context.Context, client *marketdata.Client, symbol, resolution string, from, to time.Time) ([]candle, error) {
	res := stocks.Resolution(resolution)
	raw, _, err := client.Stocks.Candles(ctx, symbol,
		stocks.WithResolution(res),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		return nil, err
	}

	candles := make([]candle, len(raw))
	for i, c := range raw {
		candles[i] = candle{
			Symbol: symbol,
			Time:   c.Time,
			Open:   c.Open,
			High:   c.High,
			Low:    c.Low,
			Close:  c.Close,
			Volume: c.Volume,
		}
	}
	return candles, nil
}

// fetchFundCandles fetches fund/ETF candles.
func fetchFundCandles(ctx context.Context, client *marketdata.Client, symbol, resolution string, from, to time.Time) ([]candle, error) {
	res := funds.Resolution(resolution)
	raw, _, err := client.Funds.Candles(ctx, symbol,
		funds.WithResolution(res),
		funds.WithCandleWindow(funds.Between(from, to)),
	)
	if err != nil {
		return nil, err
	}

	candles := make([]candle, len(raw))
	for i, c := range raw {
		candles[i] = candle{
			Symbol: symbol,
			Time:   c.Time,
			Open:   c.Open,
			High:   c.High,
			Low:    c.Low,
			Close:  c.Close,
			Volume: 0, // Funds don't have volume
		}
	}
	return candles, nil
}

// writeCSV writes candle data to a CSV file.
func writeCSV(path string, candles []candle) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	if err := w.Write([]string{"symbol", "date", "open", "high", "low", "close", "volume"}); err != nil {
		return err
	}

	for _, c := range candles {
		record := []string{
			c.Symbol,
			c.Time.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.4f", c.Open),
			fmt.Sprintf("%.4f", c.High),
			fmt.Sprintf("%.4f", c.Low),
			fmt.Sprintf("%.4f", c.Close),
			fmt.Sprintf("%d", c.Volume),
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	return nil
}
