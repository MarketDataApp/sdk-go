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

// TickersResponse represents the response structure for the /stocks/tickers API endpoint.
// It includes slices for various stock attributes such as symbol, name, type, currency, and exchange.
// Optional fields are marked with 'omitempty' to exclude them from the JSON output if they are empty.
type TickersResponse struct {
	Symbol        []string `json:"symbol"`                  // Symbol contains the stock symbols.
	Name          []string `json:"name,omitempty"`          // Name contains the names of the stocks. Optional.
	Type          []string `json:"type,omitempty"`          // Type contains the types of the stocks. Optional.
	Currency      []string `json:"currency,omitempty"`      // Currency contains the currencies in which the stocks are traded. Optional.
	Exchange      []string `json:"exchange,omitempty"`      // Exchange contains the stock exchanges on which the stocks are listed. Optional.
	FigiShares    []string `json:"figiShares,omitempty"`    // FigiShares contains the FIGI codes for the shares. Optional.
	FigiComposite []string `json:"figiComposite,omitempty"` // FigiComposite contains the composite FIGI codes. Optional.
	Cik           []string `json:"cik,omitempty"`           // Cik contains the Central Index Key (CIK) numbers. Optional.
	Updated       []int64  `json:"updated,omitempty"`       // Updated contains UNIX timestamps of the last updates. Optional.
}

// IsValid determines if the TickersResponse has at least one symbol.
//
// Returns:
//   - true if there is at least one symbol; otherwise, false.
func (tr *TickersResponse) IsValid() bool {
	return len(tr.Symbol) > 0
}

// String constructs a string representation of the TickersResponse.
// It iterates through each symbol and its associated data, formatting them into a readable string.
// If the 'Updated' field has a zero value, it prints as "nil".
//
// Returns:
//   - A string detailing the contents of the TickersResponse, including symbols, names, types, currencies, exchanges, FIGI codes, CIK numbers, and update times.
func (tr *TickersResponse) String() string {
	var str strings.Builder
	str.WriteString("TickersResponse{\n")
	for i, symbol := range tr.Symbol {
		updateTime := "nil"
		if i < len(tr.Updated) && tr.Updated[i] != 0 {
			updateTime = fmt.Sprint(tr.Updated[i])
		}
		str.WriteString(fmt.Sprintf("Ticker{Symbol: %q, Name: %q, Type: %q, Currency: %q, Exchange: %q, FigiShares: %q, FigiComposite: %q, Cik: %q, Updated: %s}\n", symbol, tr.Name[i], tr.Type[i], tr.Currency[i], tr.Exchange[i], tr.FigiShares[i], tr.FigiComposite[i], tr.Cik[i], updateTime))
	}
	str.WriteString("}")
	return str.String()
}

// Unpack converts a TickersResponse instance into a slice of Ticker structs.
//
// This method iterates through the fields of the TickersResponse struct, creating a Ticker struct for each symbol present in the response. It ensures that all corresponding fields are populated correctly. It converts the UNIX timestamp to a time.Time object for each Ticker struct if the 'Updated' field is present.
//
// Returns:
//   - A slice of Ticker structs representing the unpacked tickers.
//   - An error if the TickersResponse is nil or if there is a mismatch in the lengths of the slices within the TickersResponse.
func (tr *TickersResponse) Unpack() ([]Ticker, error) {
	if tr == nil {
		return nil, fmt.Errorf("TickersResponse is nil")
	}
	var tickerInfos []Ticker
	for i := range tr.Symbol {
		tickerInfo := Ticker{
			Symbol:        tr.Symbol[i],
			Name:          safeIndex(tr.Name, i),
			Type:          safeIndex(tr.Type, i),
			Currency:      safeIndex(tr.Currency, i),
			Exchange:      safeIndex(tr.Exchange, i),
			FigiShares:    safeIndex(tr.FigiShares, i),
			FigiComposite: safeIndex(tr.FigiComposite, i),
			Cik:           safeIndex(tr.Cik, i),
		}
		if len(tr.Updated) > i {
			tickerInfo.Updated = time.Unix(tr.Updated[i], 0)
		} else {
			tickerInfo.Updated = time.Time{} // Assign zero value of time.Time if Updated is not present
		}
		tickerInfos = append(tickerInfos, tickerInfo)
	}
	return tickerInfos, nil
}

// safeIndex safely retrieves the string at index i from the slice, or returns an empty string if out of range.
func safeIndex(slice []string, i int) string {
	if i < len(slice) {
		return slice[i]
	}
	return ""
}

// UniqueSymbols extracts and returns a slice of unique stock symbols from the TickersResponse.
//
// Returns:
//   - []string: A slice of unique stock symbols.
//   - error: An error encountered during the conversion to a map, if any.
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

// ToMap converts a TickersResponse into a map where each key is a stock symbol and its value is the corresponding Ticker struct.
//
// Returns:
//   - map[string]Ticker: A map with stock symbols as keys and Ticker structs as values.
//   - error: An error if the conversion fails.
func (tr *TickersResponse) ToMap() (map[string]Ticker, error) {
	tickerInfos, err := tr.Unpack()
	if err != nil {
		return nil, err
	}

	tickerMap := make(map[string]Ticker)
	for _, tickerInfo := range tickerInfos {
		tickerMap[tickerInfo.Symbol] = tickerInfo
	}
	return tickerMap, nil
}

// MarshalJSON customizes the JSON encoding for the TickersResponse struct.
//
// Returns:
//   - []byte: The JSON-encoded representation of the TickersResponse.
//   - error: An error if the encoding fails.
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

// Ticker represents the information of a ticker.
type Ticker struct {
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name,omitempty"`
	Type          string    `json:"type,omitempty"`
	Currency      string    `json:"currency,omitempty"`
	Exchange      string    `json:"exchange,omitempty"`
	FigiShares    string    `json:"figiShares,omitempty"`
	FigiComposite string    `json:"figiComposite,omitempty"`
	Cik           string    `json:"cik,omitempty"`
	Updated       time.Time `json:"updated,omitempty"`
}

// String generates a string representation of the Ticker struct.
//
// Returns:
//   - A string detailing the Ticker's properties in a readable format.
func (ti Ticker) String() string {
	updated := "nil"
	if !ti.Updated.IsZero() {
		updated = formatTime(ti.Updated)
	}
	return fmt.Sprintf("Ticker{Symbol: %s, Name: %s, Type: %s, Currency: %s, Exchange: %s, FigiShares: %s, FigiComposite: %s, Cik: %s, Updated: %s}", ti.Symbol, ti.Name, ti.Type, ti.Currency, ti.Exchange, ti.FigiShares, ti.FigiComposite, ti.Cik, updated)
}

// MapToTickersResponse converts a map of Ticker structs into a TickersResponse struct.
//
// Parameters:
//   - tickerMap: A map where the key is a string representing the ticker symbol, and the value is a Ticker struct.
//
// Returns:
//   - A pointer to a TickersResponse struct that aggregates the information from the input map.
func MapToTickersResponse(tickerMap map[string]Ticker) *TickersResponse {
	var tr TickersResponse
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
		tr.Updated = append(tr.Updated, tickerInfo.Updated.Unix())
	}
	return &tr
}

// SaveToCSV writes the contents of tickerMap to a CSV file specified by filename.
//
// Parameters:
//   - tickerMap: A map where the key is a string representing the ticker symbol, and the value is a Ticker struct.
//   - filename: The name of the file to which the CSV data will be written.
//
// Returns:
//   - An error if writing to the CSV file fails, otherwise nil.
func SaveToCSV(tickerMap map[string]Ticker, filename string) error {
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
		if !tickerInfo.Updated.IsZero() {
			updated = fmt.Sprintf("%v", tickerInfo.Updated.Unix())
		}
		err = writer.Write([]string{tickerInfo.Symbol, tickerInfo.Name, tickerInfo.Type, tickerInfo.Currency, tickerInfo.Exchange, tickerInfo.FigiShares, tickerInfo.FigiComposite, tickerInfo.Cik, updated})
		if err != nil {
			return err
		}
	}

	return nil
}

// CombineTickerResponses combines multiple TickersResponses into a single map.
//
// This function takes a slice of pointers to TickersResponse structs and combines them into a single map
// where the key is a string representing the ticker symbol and the value is a Ticker struct. It handles
// concurrent access to the map using a mutex to ensure thread safety.
//
// Parameters:
//   - responses: A slice of pointers to TickersResponse structs that are to be combined.
//
// Returns:
//   - A map where the key is a string representing the ticker symbol and the value is a Ticker struct.
//   - An error if any of the TickersResponse structs cannot be converted to a map or if any other error occurs during the process.
func CombineTickerResponses(responses []*TickersResponse) (map[string]Ticker, error) {
	tickerMap := make(map[string]Ticker)
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
