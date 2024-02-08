package models

import (
	"fmt"
	"strings"
	"time"
)

// StockNewsResponse represents the JSON response structure for stock news.
// It includes slices for symbols, headlines, content, sources, and publication dates,
// as well as a timestamp for when the data was last updated.
type StockNewsResponse struct {
	Symbol          []string `json:"symbol"`          // Symbol contains the stock symbols associated with the news.
	Headline        []string `json:"headline"`        // Headline contains the headlines of the news articles.
	Content         []string `json:"content"`         // Content contains the full text content of the news articles.
	Source          []string `json:"source"`          // Source contains the sources from which the news articles were obtained.
	PublicationDate []int64  `json:"publicationDate"` // PublicationDate contains the publication dates of the news articles as UNIX timestamps.
	Updated         int64    `json:"updated"`         // Updated contains the timestamp of the last update to the news data.
}

// StockNews represents a single news article related to a stock.
// It includes the stock symbol, headline, content, source, and publication date.
type StockNews struct {
	Symbol          string    // Symbol is the stock symbol associated with the news article.
	Headline        string    // Headline is the headline of the news article.
	Content         string    // Content is the full text content of the news article.
	Source          string    // Source is the source from which the news article was obtained.
	PublicationDate time.Time // PublicationDate is the publication date of the news article.
}

// String returns a formatted string representation of the StockNews struct.
//
// This method formats the StockNews fields into a readable string. If the publication date is at the start of the day,
// it prints out the date in YYYY-MM-DD format without any time or timezone info.
//
// Returns:
//   - A string representation of the StockNews struct.
func (sn StockNews) String() string {
	return fmt.Sprintf("StockNews{Symbol: %q, Headline: %q, Content: %q, Source: %q, PublicationDate: %q}",
		sn.Symbol, sn.Headline, sn.Content, sn.Source, formatTime(sn.PublicationDate))
}

// Unpack transforms the StockNewsResponse struct into a slice of StockNews structs.
//
// Returns:
//   - A slice of StockNews structs representing the unpacked news articles.
//   - An error if any issues occur during the unpacking process. This implementation always returns nil for error.
func (snr *StockNewsResponse) Unpack() ([]StockNews, error) {
	var stockNews []StockNews
	for i := range snr.Symbol {
		news := StockNews{
			Symbol:          snr.Symbol[i],
			Headline:        snr.Headline[i],
			Content:         snr.Content[i],
			Source:          snr.Source[i],
			PublicationDate: time.Unix(snr.PublicationDate[i], 0),
		}
		stockNews = append(stockNews, news)
	}
	return stockNews, nil
}

// String generates a string representation of the StockNewsResponse struct.
//
// This method iterates over each news article in the StockNewsResponse, appending a formatted string
// for each article to a strings.Builder. If the 'Updated' field is non-zero, it appends an "Updated" field
// at the end of the string.
//
// Returns:
//   - A string representation of the StockNewsResponse struct.
func (snr *StockNewsResponse) String() string {
	var result strings.Builder

	result.WriteString("StockNewsResponse{")
	for i := range snr.Symbol {
		fmt.Fprintf(&result, "Symbol: %q, Headline: %q, Content: %q, Source: %q, PublicationDate: %v",
			snr.Symbol[i], snr.Headline[i], snr.Content[i], snr.Source[i], snr.PublicationDate[i])

		if i < len(snr.Symbol)-1 {
			result.WriteString("; ")
		}
	}

	if snr.Updated != 0 {
		fmt.Fprintf(&result, ", Updated: %v", snr.Updated)
	}

	result.WriteString("}")
	return result.String()
}
