package models

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/iancoleman/orderedmap"
)

// TickersResponse represents the response from the /stocks/tickers endpoint.
type TickersResponse struct {
	Symbol        []string `json:"symbol"`
	Name          []string `json:"name,omitempty"`
	Type          []string `json:"type,omitempty"`
	Currency      []string `json:"currency,omitempty"`
	Exchange      []string `json:"exchange,omitempty"`
	FigiShares    []string `json:"figiShares,omitempty"`
	FigiComposite []string `json:"figiComposite,omitempty"`
	Cik           []string `json:"cik,omitempty"`
	Updated       *[]int64 `json:"updated,omitempty"`
}

// IsValid checks if the TickersResponse is valid.
func (tr *TickersResponse) IsValid() bool {
	return len(tr.Symbol) > 0
}

// String returns a string representation of TickersResponse.
func (tr *TickersResponse) String() string {
	var str strings.Builder
	str.WriteString("TickersResponse{\n")
	for i := range tr.Symbol {
		str.WriteString(fmt.Sprintf("Symbol: %s, Name: %s, Type: %s, Currency: %s, Exchange: %s, FigiShares: %s, FigiComposite: %s, Cik: %s\n", tr.Symbol[i], tr.Name[i], tr.Type[i], tr.Currency[i], tr.Exchange[i], tr.FigiShares[i], tr.FigiComposite[i], tr.Cik[i]))
		if tr.Updated != nil && i < len(*tr.Updated) && (*tr.Updated)[i] != 0 {
			str.WriteString(fmt.Sprintf("Updated: %v\n", time.Unix((*tr.Updated)[i], 0)))
		} else {
			str.WriteString("Updated: \n")
		}
	}
	str.WriteString("}")
	return str.String()
}

// Unpack converts TickersResponse to a slice of TickerObj.
func (tr *TickersResponse) Unpack() ([]TickerObj, error) {
	if tr == nil || tr.Updated == nil {
		return nil, fmt.Errorf("TickersResponse or its Updated field is nil")
	}
	var tickerInfos []TickerObj
	for i := range tr.Symbol {
		if i >= len(tr.Name) || i >= len(tr.Type) || i >= len(tr.Currency) || i >= len(tr.Exchange) || i >= len(tr.FigiShares) || i >= len(tr.FigiComposite) || i >= len(tr.Cik) || (tr.Updated != nil && i >= len(*tr.Updated)) {
			return nil, fmt.Errorf("index out of range")
		}
		tickerInfo := TickerObj{
			Symbol:        tr.Symbol[i],
			Name:          tr.Name[i],
			Type:          tr.Type[i],
			Currency:      tr.Currency[i],
			Exchange:      tr.Exchange[i],
			FigiShares:    tr.FigiShares[i],
			FigiComposite: tr.FigiComposite[i],
			Cik:           tr.Cik[i],
		}
		if tr.Updated != nil && (*tr.Updated)[i] != 0 {
			t := time.Unix((*tr.Updated)[i], 0)
			tickerInfo.Updated = &t
		}
		tickerInfos = append(tickerInfos, tickerInfo)
	}
	return tickerInfos, nil
}

// UniqueSymbols extracts and returns a slice of unique stock symbols from the TickersResponse.
// It returns the slice of unique symbols and any error encountered during the conversion to a map.
func (tr *TickersResponse) UniqueSymbols() ([]string, error) {
	tickerMap, err := tr.ToMap()
	if err != nil {
		return nil, err
	}

	uniqueSymbols := make([]string, 0, len(tickerMap))
	for symbol := range tickerMap {
		uniqueSymbols = append(uniqueSymbols, symbol)
	}

	return uniqueSymbols, nil
}

// ToMap converts TickersResponse to a map with the symbol as the key.
func (tr *TickersResponse) ToMap() (map[string]TickerObj, error) {
	tickerInfos, err := tr.Unpack()
	if err != nil {
		return nil, err
	}

	tickerMap := make(map[string]TickerObj)
	for _, tickerInfo := range tickerInfos {
		tickerMap[tickerInfo.Symbol] = tickerInfo
	}
	return tickerMap, nil
}

// MarshalJSON is a method on the TickersResponse struct.
func (tr *TickersResponse) MarshalJSON() ([]byte, error) {
	if tr == nil {
		return nil, fmt.Errorf("TickersResponse is nil")
	}
	// Create a new ordered map
	o := orderedmap.New()

	// Set the "s" key to "ok"
	o.Set("s", "ok")

	// Set the other keys to the corresponding slices in the struct
	o.Set("symbol", tr.Symbol)
	o.Set("name", tr.Name)
	o.Set("type", tr.Type)
	o.Set("currency", tr.Currency)
	o.Set("exchange", tr.Exchange)
	o.Set("figiShares", tr.FigiShares)
	o.Set("figiComposite", tr.FigiComposite)
	o.Set("cik", tr.Cik)
	o.Set("updated", tr.Updated)

	// Marshal the ordered map into a JSON object and return the result
	return json.Marshal(o)
}

// TickerObj represents the information of a ticker.
type TickerObj struct {
	Symbol        string
	Name          string
	Type          string
	Currency      string
	Exchange      string
	FigiShares    string
	FigiComposite string
	Cik           string
	Updated       *time.Time
}

// String returns a string representation of TickerObj.
func (ti *TickerObj) String() string {
	updated := ""
	if ti.Updated != nil {
		updated = ti.Updated.String()
	}
	return fmt.Sprintf("TickerObj{Symbol: %s, Name: %s, Type: %s, Currency: %s, Exchange: %s, FigiShares: %s, FigiComposite: %s, Cik: %s, Updated: %s}", ti.Symbol, ti.Name, ti.Type, ti.Currency, ti.Exchange, ti.FigiShares, ti.FigiComposite, ti.Cik, updated)
}

// MapToTickersResponse converts a map of TickerObj to a TickersResponse.
func MapToTickersResponse(tickerMap map[string]TickerObj) *TickersResponse {
	var tr TickersResponse
	tr.Updated = new([]int64) // Initialize tr.Updated
	keys := make([]string, 0, len(tickerMap))
	for key := range tickerMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		tickerInfo := tickerMap[key]
		tr.Symbol = append(tr.Symbol, tickerInfo.Symbol)
		tr.Name = append(tr.Name, tickerInfo.Name)
		tr.Type = append(tr.Type, tickerInfo.Type)
		tr.Currency = append(tr.Currency, tickerInfo.Currency)
		tr.Exchange = append(tr.Exchange, tickerInfo.Exchange)
		tr.FigiShares = append(tr.FigiShares, tickerInfo.FigiShares)
		tr.FigiComposite = append(tr.FigiComposite, tickerInfo.FigiComposite)
		tr.Cik = append(tr.Cik, tickerInfo.Cik)
		if tickerInfo.Updated != nil {
			*tr.Updated = append(*tr.Updated, tickerInfo.Updated.Unix())
		} else {
			*tr.Updated = append(*tr.Updated, 0)
		}
	}
	return &tr
}

// SaveToCSV saves the ticker map to a CSV file.
func SaveToCSV(tickerMap map[string]TickerObj, filename string) error {
	if tickerMap == nil {
		return fmt.Errorf("tickerMap is nil")
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	err = writer.Write([]string{"Symbol", "Name", "Type", "Currency", "Exchange", "FigiShares", "FigiComposite", "Cik", "Updated"})
	if err != nil {
		return err
	}

	// Write data
	for _, tickerInfo := range tickerMap {
		updated := ""
		if tickerInfo.Updated != nil {
			updated = tickerInfo.Updated.String()
		}
		err = writer.Write([]string{tickerInfo.Symbol, tickerInfo.Name, tickerInfo.Type, tickerInfo.Currency, tickerInfo.Exchange, tickerInfo.FigiShares, tickerInfo.FigiComposite, tickerInfo.Cik, updated})
		if err != nil {
			return err
		}
	}

	return nil
}

// CombineTickerResponses combines multiple TickersResponses into a single map.
func CombineTickerResponses(responses []*TickersResponse) (map[string]TickerObj, error) {
	tickerMap := make(map[string]TickerObj)
	var mutex sync.Mutex

	var wg sync.WaitGroup
	errors := make(chan error)

	for _, response := range responses {
		wg.Add(1)
		go func(response *TickersResponse) {
			defer wg.Done()
			responseMap, err := response.ToMap()
			if err != nil {
				errors <- err
				return
			}
			mutex.Lock()
			for key, value := range responseMap {
				tickerMap[key] = value
			}
			mutex.Unlock()
		}(response)
	}

	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		if err != nil {
			return nil, err
		}
	}

	return tickerMap, nil
}
