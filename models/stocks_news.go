package models

import (
	"fmt"
	"strings"
	"time"
)

type StockNewsResponse struct {
	Symbol          []string `json:"symbol"`
	Headline        []string `json:"headline"`
	Content         []string `json:"content"`
	Source          []string `json:"source"`
	PublicationDate []int64  `json:"publicationDate"`
	Updated         int64    `json:"updated"`
}

type StockNews struct {
	Symbol          string
	Headline        string
	Content         string
	Source          string
	PublicationDate time.Time
}

func (sn StockNews) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	return fmt.Sprintf("Symbol: %s, Headline: %s, Content: %s, Source: %s, PublicationDate: %s",
		sn.Symbol, sn.Headline, sn.Content, sn.Source, sn.PublicationDate.In(loc).Format("2006-01-02 15:04:05 Z07:00"))
}

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

func (snr *StockNewsResponse) String() string {
	var result strings.Builder

	for i := range snr.Symbol {
		fmt.Fprintf(&result, "Symbol: %s, Headline: %s, Content: %s, Source: %s, Publication Date: %v",
			snr.Symbol[i], snr.Headline[i], snr.Content[i], snr.Source[i], snr.PublicationDate[i])

		if i < len(snr.Symbol)-1 {
			result.WriteString("; ")
		}
	}

	if snr.Updated != 0 {
		fmt.Fprintf(&result, ", Updated: %v", snr.Updated)
	}

	return result.String()
}
