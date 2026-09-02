// Example: Earnings Analyzer
//
// Concurrent data pipeline that fetches earnings data for a watchlist,
// correlates each earnings report with the stock's price reaction using
// candle data, and optionally fetches related news headlines. Outputs a
// summary report ranked by earnings surprise magnitude.
//
// Usage:
//
//	export MARKETDATA_TOKEN="your-api-key"
//	go run ./examples/earnings-analyzer
//	go run ./examples/earnings-analyzer -symbols AAPL,MSFT,GOOG -quarters 4
//	go run ./examples/earnings-analyzer -symbols TSLA -quarters 8 -news
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// earningsReport combines earnings data with price reaction and news.
type earningsReport struct {
	Symbol        string
	FiscalQuarter int
	FiscalYear    int
	ReportDate    time.Time
	ReportedEPS   *float64
	EstimatedEPS  *float64
	SurpriseEPS   *float64
	SurprisePct   *float64

	// Price reaction (calculated from candles)
	PriceBeforeClose float64 // Close on the day before earnings
	PriceAfterClose  float64 // Close on the day after earnings
	PriceReaction    float64 // % change between the two closes
	ReactionBullish  bool    // sign of PriceReaction, not the after-day candle's own direction

	// News headlines (optional)
	Headlines []string

	// Data availability
	HasPriceData bool
	HasNews      bool
}

func main() {
	symbolsFlag := flag.String("symbols", "AAPL,MSFT,GOOG,AMZN,META", "Comma-separated list of stock symbols")
	quartersFlag := flag.Int("quarters", 4, "Number of past earnings quarters to analyze")
	newsFlag := flag.Bool("news", false, "Also fetch news headlines around earnings dates")
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	fmt.Printf("Earnings Analyzer — %d symbols, last %d quarters\n\n", len(symbols), *quartersFlag)

	start := time.Now()

	// Concurrent fetch: one goroutine per symbol
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allReports []earningsReport

	for _, sym := range symbols {
		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()

			reports, analyzeErr := analyzeSymbol(ctx, client, symbol, *quartersFlag, *newsFlag)
			if analyzeErr != nil {
				log.Printf("[%s] Error: %v", symbol, analyzeErr)
				return
			}

			mu.Lock()
			allReports = append(allReports, reports...)
			mu.Unlock()

			fmt.Printf("  [%s] %d earnings reports analyzed\n", symbol, len(reports))
		}(sym)
	}
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Printf("\nAnalysis complete in %s\n", elapsed.Round(time.Millisecond))

	if len(allReports) == 0 {
		fmt.Println("No earnings data found.")
		return
	}

	// Sort by absolute surprise percentage (biggest surprises first)
	sort.Slice(allReports, func(i, j int) bool {
		si := absPtr(allReports[i].SurprisePct)
		sj := absPtr(allReports[j].SurprisePct)
		return si > sj
	})

	// Display report
	fmt.Printf("\nEarnings Report (%d results):\n\n", len(allReports))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "SYMBOL\tQTR\tREPORT DATE\tREPORTED\tESTIMATED\tSURPRISE\tSURPRISE%\tPRICE REACTION\t")
	fmt.Fprintln(w, "------\t---\t-----------\t--------\t---------\t--------\t---------\t--------------\t")

	for _, r := range allReports {
		reported := "n/a"
		if r.ReportedEPS != nil {
			reported = fmt.Sprintf("$%.2f", *r.ReportedEPS)
		}
		estimated := "n/a"
		if r.EstimatedEPS != nil {
			estimated = fmt.Sprintf("$%.2f", *r.EstimatedEPS)
		}
		surprise := "n/a"
		if r.SurpriseEPS != nil {
			prefix := ""
			if *r.SurpriseEPS > 0 {
				prefix = "+"
			}
			surprise = fmt.Sprintf("%s$%.2f", prefix, *r.SurpriseEPS)
		}
		surprisePct := "n/a"
		if r.SurprisePct != nil {
			prefix := ""
			if *r.SurprisePct > 0 {
				prefix = "+"
			}
			surprisePct = fmt.Sprintf("%s%.1f%%", prefix, *r.SurprisePct*100)
		}

		priceReaction := "n/a"
		if r.HasPriceData {
			prefix := ""
			if r.PriceReaction > 0 {
				prefix = "+"
			}
			direction := "bearish"
			if r.ReactionBullish {
				direction = "bullish"
			}
			priceReaction = fmt.Sprintf("%s%.2f%% (%s)", prefix, r.PriceReaction, direction)
		}

		fmt.Fprintf(w, "%s\tQ%d %d\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
			r.Symbol,
			r.FiscalQuarter, r.FiscalYear,
			r.ReportDate.Format("2006-01-02"),
			reported,
			estimated,
			surprise,
			surprisePct,
			priceReaction,
		)
	}
	w.Flush()

	// Show news headlines if requested
	if *newsFlag {
		fmt.Println("\n--- Headlines around earnings dates ---")
		for _, r := range allReports {
			if len(r.Headlines) > 0 {
				fmt.Printf("\n%s Q%d %d (%s):\n", r.Symbol, r.FiscalQuarter, r.FiscalYear, r.ReportDate.Format("2006-01-02"))
				for _, h := range r.Headlines {
					fmt.Printf("  - %s\n", h)
				}
			}
		}
	}

	// Summary statistics
	var beats, misses, inLine int
	var totalReaction float64
	var priceCount int
	for _, r := range allReports {
		if r.SurpriseEPS != nil {
			if *r.SurpriseEPS > 0 {
				beats++
			} else if *r.SurpriseEPS < 0 {
				misses++
			} else {
				inLine++
			}
		}
		if r.HasPriceData {
			totalReaction += math.Abs(r.PriceReaction)
			priceCount++
		}
	}

	fmt.Printf("\nSummary: %d beats, %d misses, %d in-line", beats, misses, inLine)
	if priceCount > 0 {
		fmt.Printf(" | Avg absolute price reaction: %.2f%%", totalReaction/float64(priceCount))
	}
	fmt.Println()

	// client.RateLimits() is a snapshot of the LAST completed response, not a
	// running total: Consumed is what that single request cost. A session
	// total means summing resp.RateLimit.Consumed across your own calls — see
	// the response-formats example.
	limits := client.RateLimits()
	fmt.Printf("Credits remaining: %d (last request cost %d)\n", limits.Remaining, limits.Consumed)
}

// analyzeSymbol fetches earnings, candles, and optionally news for one symbol.
func analyzeSymbol(ctx context.Context, client *marketdata.Client, symbol string, quarters int, fetchNews bool) ([]earningsReport, error) {
	// Fetch earnings using a date range to get historical data.
	// The countback parameter returns upcoming earnings, so we use from/to
	// to look back into the past.
	lookbackYears := (quarters / 4) + 1
	from := time.Now().AddDate(-lookbackYears, 0, 0)
	to := time.Now()

	earnings, _, err := client.Stocks.Earnings(ctx, symbol,
		stocks.WithEarningsWindow(stocks.Between(from, to)),
	)
	if err != nil {
		return nil, err
	}

	// Limit to the requested number of quarters (take the most recent)
	if len(earnings) > quarters {
		earnings = earnings[len(earnings)-quarters:]
	}

	var reports []earningsReport
	for _, e := range earnings {
		report := earningsReport{
			Symbol:        e.Symbol,
			FiscalQuarter: e.FiscalQuarter,
			FiscalYear:    e.FiscalYear,
			ReportDate:    e.ReportDate,
			ReportedEPS:   e.ReportedEPS,
			EstimatedEPS:  e.EstimatedEPS,
			SurpriseEPS:   e.SurpriseEPS,
			SurprisePct:   e.SurpriseEPSPercent,
		}

		// Skip future earnings (no reported EPS yet)
		if e.ReportedEPS == nil {
			reports = append(reports, report)
			continue
		}

		// Fetch candles around the earnings date to measure price reaction
		// Get 5 trading days around the report date
		from := e.ReportDate.AddDate(0, 0, -3)
		to := e.ReportDate.AddDate(0, 0, 5)

		candles, _, candleErr := client.Stocks.Candles(ctx, symbol,
			stocks.WithResolution(stocks.ResolutionDaily),
			stocks.WithCandleWindow(stocks.Between(from, to)),
		)
		if candleErr == nil && len(candles) >= 2 {
			// Find the candle before and after the report date
			var before, after *stocks.Candle
			for i := range candles {
				c := &candles[i]
				if c.Time.Before(e.ReportDate) || c.Time.Equal(e.ReportDate) {
					before = c
				}
				if c.Time.After(e.ReportDate) && after == nil {
					after = c
				}
			}

			if before != nil && after != nil && before.Close > 0 {
				report.PriceBeforeClose = before.Close
				report.PriceAfterClose = after.Close
				report.PriceReaction = ((after.Close - before.Close) / before.Close) * 100
				// Label the reaction the table actually prints. Candle.IsBullish
				// asks whether the after day closed above its OWN open, which is
				// an intraday direction unrelated to the earnings gap — using it
				// here printed rows like "+9.58% (bearish)".
				report.ReactionBullish = report.PriceReaction > 0
				report.HasPriceData = true
			}
		}

		// Fetch news around the earnings date (optional)
		if fetchNews {
			news, _, newsErr := client.Stocks.News(ctx, symbol,
				stocks.WithNewsWindow(stocks.Between(e.ReportDate.AddDate(0, 0, -1), e.ReportDate.AddDate(0, 0, 2))),
			)
			if newsErr == nil && len(news) > 0 {
				for _, n := range news {
					if len(report.Headlines) < 3 { // Limit to 3 headlines
						report.Headlines = append(report.Headlines, n.Headline)
					}
				}
				report.HasNews = true
			}
		}

		reports = append(reports, report)
	}

	return reports, nil
}

func absPtr(f *float64) float64 {
	if f == nil {
		return 0
	}
	return math.Abs(*f)
}
