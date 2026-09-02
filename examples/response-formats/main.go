// Command response-formats demonstrates the *Response value every context-first
// SDK method returns alongside its typed data — the part most examples discard
// with a blank identifier.
//
// It covers four things the other examples never touch:
//
//   - the CSV facet (client.Stocks.AsCSV() and friends), which asks the API for
//     format=csv and hands back the raw text
//   - the format predicates (IsJSON / IsCSV / IsHTML)
//   - Response.Body(), the raw payload as the API sent it
//   - Response.SaveToFile(), which writes that payload straight to disk
//
// It also shows the correct way to total API credits for a session: sum
// resp.RateLimit.Consumed across your own calls. client.RateLimits() is a
// snapshot of the last completed response, so its Consumed field is what that
// single request cost — not a running total. The other examples' footers say
// so and point here.
//
// Usage:
//
//	go run ./examples/response-formats [-symbol AAPL] [-outdir /tmp/md]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// credits accumulates a real session total, one response at a time. This is
// the pattern client.RateLimits() cannot give you: its Consumed field reports
// only the most recent request's cost, and under concurrency it reports
// whichever request happened to finish last.
type credits struct {
	total atomic.Int64
}

// add records what one response cost. It tolerates a nil response so callers
// can feed it a no-data result without branching.
func (c *credits) add(resp *marketdata.Response) {
	if resp == nil {
		return
	}
	c.total.Add(int64(resp.RateLimit.Consumed))
}

func (c *credits) String() string {
	return fmt.Sprintf("%d credits", c.total.Load())
}

func main() {
	symbol := flag.String("symbol", "AAPL", "Stock symbol to fetch")
	outdir := flag.String("outdir", "", "Directory to write saved payloads (default: a temp dir)")
	flag.Parse()

	dir := *outdir
	if dir == "" {
		var err error
		dir, err = os.MkdirTemp("", "marketdata-formats-")
		if err != nil {
			log.Fatalf("create temp dir: %v", err)
		}
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("create output directory: %v", err)
	}

	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	spent := &credits{}

	fmt.Printf("Response Formats — %s\n", *symbol)
	fmt.Println(strings.Repeat("=", 60))

	typedJSON(ctx, client, *symbol, spent, dir)
	rawCSV(ctx, client, *symbol, spent, dir)

	fmt.Printf("\nSession total: %s (summed from each response, not from client.RateLimits())\n", spent)
	if spent.total.Load() == 0 {
		// Not a broken counter: the API bills nothing for a cached answer,
		// which it signals with HTTP 203 Non-Authoritative Information.
		fmt.Println("  (zero because the API served these from cache — HTTP 203 costs no credits)")
	}
	fmt.Printf("Credits remaining: %d\n", client.RateLimits().Remaining)
	fmt.Printf("Files written to: %s\n", dir)
}

// typedJSON shows the default path: typed structs, plus the metadata that rides
// along on the *Response.
func typedJSON(ctx context.Context, client *marketdata.Client, symbol string, spent *credits, dir string) {
	fmt.Println("\n--- TYPED JSON (the default) ---")

	candles, resp, err := client.Stocks.Candles(ctx, symbol,
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.LastN(5)),
	)
	spent.add(resp)
	if err != nil {
		fmt.Printf("  error: %v\n", err)
		return
	}
	if resp != nil && resp.NoData {
		fmt.Println("  no data for this request")
		return
	}

	fmt.Printf("  %d candles decoded into []stocks.Candle\n", len(candles))
	if len(candles) > 0 {
		// Every public type implements String() (requirements §11.6).
		fmt.Printf("  latest: %s\n", candles[len(candles)-1])
	}

	// The format predicates describe what the server actually sent.
	fmt.Printf("  IsJSON=%t IsCSV=%t IsHTML=%t\n", resp.IsJSON(), resp.IsCSV(), resp.IsHTML())

	// Status and headers come from the embedded *http.Response.
	fmt.Printf("  status=%s  this request cost %d credit(s), %d remain\n",
		resp.Status, resp.RateLimit.Consumed, resp.RateLimit.Remaining)

	// Body() returns the payload exactly as the API sent it — a copy, so
	// mutating it cannot disturb a later SaveToFile.
	body := resp.Body()
	fmt.Printf("  raw body: %d bytes, starts %q\n", len(body), preview(body, 48))

	path := filepath.Join(dir, symbol+"_candles.json")
	if err := resp.SaveToFile(path); err != nil {
		fmt.Printf("  save failed: %v\n", err)
		return
	}
	fmt.Printf("  saved to %s\n", filepath.Base(path))
}

// rawCSV shows the CSV facet: the same endpoint, asked for format=csv, handed
// back as the API's own text rather than parsed into rows.
func rawCSV(ctx context.Context, client *marketdata.Client, symbol string, spent *credits, dir string) {
	fmt.Println("\n--- RAW CSV (the AsCSV facet) ---")

	// The CSV facet returns a *response.CSVResponse, which embeds the same
	// Response the JSON path returns — so the metadata, predicates, Body()
	// and SaveToFile() are all still there.
	csv, err := client.Stocks.AsCSV().Candles(ctx, symbol,
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.LastN(5)),
	)
	if csv != nil {
		spent.add(&csv.Response)
	}
	if err != nil {
		fmt.Printf("  error: %v\n", err)
		return
	}
	if csv == nil {
		fmt.Println("  no CSV returned")
		return
	}

	// The facet deliberately does not parse rows: the API's text is the
	// product. IsCSV() is true here and false on the JSON call above.
	fmt.Printf("  IsJSON=%t IsCSV=%t IsHTML=%t\n", csv.IsJSON(), csv.IsCSV(), csv.IsHTML())

	lines := strings.Split(strings.TrimRight(csv.CSV(), "\n"), "\n")
	fmt.Printf("  %d lines of CSV text\n", len(lines))
	for i, line := range lines {
		if i >= 3 {
			fmt.Printf("  ... %d more\n", len(lines)-3)
			break
		}
		fmt.Printf("  | %s\n", line)
	}

	path := filepath.Join(dir, symbol+"_candles.csv")
	if err := csv.SaveToFile(path); err != nil {
		fmt.Printf("  save failed: %v\n", err)
		return
	}
	fmt.Printf("  saved to %s\n", filepath.Base(path))
}

// preview returns the first n bytes of b as a single-line string.
func preview(b []byte, n int) string {
	s := strings.ReplaceAll(string(b), "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
