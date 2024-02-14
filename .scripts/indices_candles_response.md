




<a name="ByClose"></a>
## type ByClose

```go
type ByClose []Candle
```

ByClose implements sort.Interface for \[\]Candle based on the Close field.



<Tabs>
<TabItem value="Example" label="Example">




```go
// Create a slice of Candle instances
candles := []Candle{
	{Symbol: "AAPL", Date: time.Now(), Open: 100, High: 105, Low: 95, Close: 102, Volume: 1000},
	{Symbol: "AAPL", Date: time.Now(), Open: 102, High: 106, Low: 98, Close: 104, Volume: 1500},
	{Symbol: "AAPL", Date: time.Now(), Open: 99, High: 103, Low: 97, Close: 100, Volume: 1200},
}

// Sort the candles by their Close value using sort.Sort and ByClose
sort.Sort(ByClose(candles))

// Print the sorted candles to demonstrate the order
for _, candle := range candles {
	fmt.Printf("Close: %v\n", candle.Close)
}

```

#### Output

```
Close: 100
Close: 102
Close: 104
```

</TabItem>
</Tabs>

<a name="ByClose.Len"></a>
### func \(ByClose\) Len

```go
func (a ByClose) Len() int
```



<a name="ByClose.Less"></a>
### func \(ByClose\) Less

```go
func (a ByClose) Less(i, j int) bool
```



<a name="ByClose.Swap"></a>
### func \(ByClose\) Swap

```go
func (a ByClose) Swap(i, j int)
```



<a name="ByDate"></a>
## type ByDate

```go
type ByDate []Candle
```

ByDate implements sort.Interface for \[\]Candle based on the Date field. This allows for sorting a slice of Candle instances by their Date field in ascending order.



<Tabs>
<TabItem value="Example" label="Example">




```go
// Assuming the Candle struct has at least a Date field of type time.Time
candles := []Candle{
	{Date: time.Date(2023, 3, 10, 0, 0, 0, 0, time.UTC)},
	{Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
	{Date: time.Date(2023, 2, 20, 0, 0, 0, 0, time.UTC)},
}

// Sorting the slice of Candle instances by their Date field in ascending order
sort.Sort(ByDate(candles))

// Printing out the sorted dates to demonstrate the order
for _, candle := range candles {
	fmt.Println(candle.Date.Format("2006-01-02"))
}

```

#### Output

```
2023-01-05
2023-02-20
2023-03-10
```

</TabItem>
</Tabs>

<a name="ByDate.Len"></a>
### func \(ByDate\) Len

```go
func (a ByDate) Len() int
```



<a name="ByDate.Less"></a>
### func \(ByDate\) Less

```go
func (a ByDate) Less(i, j int) bool
```



<a name="ByDate.Swap"></a>
### func \(ByDate\) Swap

```go
func (a ByDate) Swap(i, j int)
```



<a name="ByHigh"></a>
## type ByHigh

```go
type ByHigh []Candle
```

ByHigh implements sort.Interface for \[\]Candle based on the High field.



<Tabs>
<TabItem value="Example" label="Example">




```go
// Assuming the Candle struct has at least a High field of type float64
candles := []Candle{
	{High: 15.2},
	{High: 11.4},
	{High: 13.5},
}

// Sorting the slice of Candle instances by their High field in ascending order
sort.Sort(ByHigh(candles))

// Printing out the sorted High values to demonstrate the order
for _, candle := range candles {
	fmt.Printf("%.1f\n", candle.High)
}

```

#### Output

```
11.4
13.5
15.2
```

</TabItem>
</Tabs>

<a name="ByHigh.Len"></a>
### func \(ByHigh\) Len

```go
func (a ByHigh) Len() int
```



<a name="ByHigh.Less"></a>
### func \(ByHigh\) Less

```go
func (a ByHigh) Less(i, j int) bool
```



<a name="ByHigh.Swap"></a>
### func \(ByHigh\) Swap

```go
func (a ByHigh) Swap(i, j int)
```



<a name="ByLow"></a>
## type ByLow

```go
type ByLow []Candle
```

ByLow implements sort.Interface for \[\]Candle based on the Low field.



<Tabs>
<TabItem value="Example" label="Example">




```go
// Assuming the Candle struct has at least a Low field of type float64
candles := []Candle{
	{Low: 5.5},
	{Low: 7.2},
	{Low: 6.3},
}

// Sorting the slice of Candle instances by their Low field in ascending order
sort.Sort(ByLow(candles))

// Printing out the sorted Low values to demonstrate the order
for _, candle := range candles {
	fmt.Printf("%.1f\n", candle.Low)
}

```

#### Output

```
5.5
6.3
7.2
```

</TabItem>
</Tabs>

<a name="ByLow.Len"></a>
### func \(ByLow\) Len

```go
func (a ByLow) Len() int
```



<a name="ByLow.Less"></a>
### func \(ByLow\) Less

```go
func (a ByLow) Less(i, j int) bool
```



<a name="ByLow.Swap"></a>
### func \(ByLow\) Swap

```go
func (a ByLow) Swap(i, j int)
```



<a name="ByN"></a>
## type ByN

```go
type ByN []Candle
```

ByN implements sort.Interface for \[\]Candle based on the N field.



<Tabs>
<TabItem value="Example" label="Example">




```go
// Assuming the Candle struct has at least an N field of type int (or any comparable type)
candles := []Candle{
	{N: 3},
	{N: 1},
	{N: 2},
}

// Sorting the slice of Candle instances by their N field in ascending order
sort.Sort(ByN(candles))

// Printing out the sorted N values to demonstrate the order
for _, candle := range candles {
	fmt.Println(candle.N)
}

```

#### Output

```
1
2
3
```

</TabItem>
</Tabs>

<a name="ByN.Len"></a>
### func \(ByN\) Len

```go
func (a ByN) Len() int
```



<a name="ByN.Less"></a>
### func \(ByN\) Less

```go
func (a ByN) Less(i, j int) bool
```



<a name="ByN.Swap"></a>
### func \(ByN\) Swap

```go
func (a ByN) Swap(i, j int)
```



<a name="ByOpen"></a>
## type ByOpen

```go
type ByOpen []Candle
```

ByOpen implements sort.Interface for \[\]Candle based on the Open field.



<Tabs>
<TabItem value="Example" label="Example">




```go
// Assuming the Candle struct has at least an Open field of type float64
candles := []Candle{
	{Open: 10.5},
	{Open: 8.2},
	{Open: 9.7},
}

// Sorting the slice of Candle instances by their Open field in ascending order
sort.Sort(ByOpen(candles))

// Printing out the sorted Open values to demonstrate the order
for _, candle := range candles {
	fmt.Printf("%.1f\n", candle.Open)
}

```

#### Output

```
8.2
9.7
10.5
```

</TabItem>
</Tabs>

<a name="ByOpen.Len"></a>
### func \(ByOpen\) Len

```go
func (a ByOpen) Len() int
```



<a name="ByOpen.Less"></a>
### func \(ByOpen\) Less

```go
func (a ByOpen) Less(i, j int) bool
```



<a name="ByOpen.Swap"></a>
### func \(ByOpen\) Swap

```go
func (a ByOpen) Swap(i, j int)
```



<a name="BySymbol"></a>
## type BySymbol

```go
type BySymbol []Candle
```

BySymbol implements sort.Interface for \[\]Candle based on the Symbol field. Candles are sorted in ascending order.



<Tabs>
<TabItem value="Example" label="Example">




```go
// Create a slice of Candle instances with different symbols
candles := []Candle{
	{Symbol: "MSFT", Date: time.Date(2023, 4, 10, 0, 0, 0, 0, time.UTC), Open: 250.0, High: 255.0, Low: 248.0, Close: 252.0, Volume: 3000},
	{Symbol: "AAPL", Date: time.Date(2023, 4, 10, 0, 0, 0, 0, time.UTC), Open: 150.0, High: 155.0, Low: 149.0, Close: 152.0, Volume: 2000},
	{Symbol: "GOOGL", Date: time.Date(2023, 4, 10, 0, 0, 0, 0, time.UTC), Open: 1200.0, High: 1210.0, Low: 1195.0, Close: 1205.0, Volume: 1000},
}

// Sort the candles by their Symbol using sort.Sort and BySymbol
sort.Sort(BySymbol(candles))

// Print the sorted candles to demonstrate the order
for _, candle := range candles {
	fmt.Printf("Symbol: %s, Close: %.2f\n", candle.Symbol, candle.Close)
}

```

#### Output

```
Symbol: AAPL, Close: 152.00
Symbol: GOOGL, Close: 1205.00
Symbol: MSFT, Close: 252.00
```

</TabItem>
</Tabs>

<a name="BySymbol.Len"></a>
### func \(BySymbol\) Len

```go
func (a BySymbol) Len() int
```



<a name="BySymbol.Less"></a>
### func \(BySymbol\) Less

```go
func (a BySymbol) Less(i, j int) bool
```



<a name="BySymbol.Swap"></a>
### func \(BySymbol\) Swap

```go
func (a BySymbol) Swap(i, j int)
```



<a name="ByVWAP"></a>
## type ByVWAP

```go
type ByVWAP []Candle
```

ByVWAP implements sort.Interface for \[\]Candle based on the VWAP field.



<Tabs>
<TabItem value="Example" label="Example">




```go
// Assuming the Candle struct has at least a VWAP (Volume Weighted Average Price) field of type float64
candles := []Candle{
	{VWAP: 10.5},
	{VWAP: 8.2},
	{VWAP: 9.7},
}

// Sorting the slice of Candle instances by their VWAP field in ascending order
sort.Sort(ByVWAP(candles))

// Printing out the sorted VWAP values to demonstrate the order
for _, candle := range candles {
	fmt.Printf("%.1f\n", candle.VWAP)
}

```

#### Output

```
8.2
9.7
10.5
```

</TabItem>
</Tabs>

<a name="ByVWAP.Len"></a>
### func \(ByVWAP\) Len

```go
func (a ByVWAP) Len() int
```



<a name="ByVWAP.Less"></a>
### func \(ByVWAP\) Less

```go
func (a ByVWAP) Less(i, j int) bool
```



<a name="ByVWAP.Swap"></a>
### func \(ByVWAP\) Swap

```go
func (a ByVWAP) Swap(i, j int)
```



<a name="ByVolume"></a>
## type ByVolume

```go
type ByVolume []Candle
```

ByVolume implements sort.Interface for \[\]Candle based on the Volume field.



<Tabs>
<TabItem value="Example" label="Example">




```go
// Assuming the Candle struct has at least a Volume field of type int
candles := []Candle{
	{Volume: 300},
	{Volume: 100},
	{Volume: 200},
}

// Sorting the slice of Candle instances by their Volume field in ascending order
sort.Sort(ByVolume(candles))

// Printing out the sorted volumes to demonstrate the order
for _, candle := range candles {
	fmt.Println(candle.Volume)
}

```

#### Output

```
100
200
300
```

</TabItem>
</Tabs>

<a name="ByVolume.Len"></a>
### func \(ByVolume\) Len

```go
func (a ByVolume) Len() int
```



<a name="ByVolume.Less"></a>
### func \(ByVolume\) Less

```go
func (a ByVolume) Less(i, j int) bool
```



<a name="ByVolume.Swap"></a>
### func \(ByVolume\) Swap

```go
func (a ByVolume) Swap(i, j int)
```



<a name="Candle"></a>
## type Candle

```go
type Candle struct {
    Symbol string    `json:"symbol,omitempty"` // The symbol of the candle.
    Date   time.Time `json:"t"`                // Date represents the date and time of the candle.
    Open   float64   `json:"o"`                // Open is the opening price of the candle.
    High   float64   `json:"h"`                // High is the highest price reached during the candle's time.
    Low    float64   `json:"l"`                // Low is the lowest price reached during the candle's time.
    Close  float64   `json:"c"`                // Close is the closing price of the candle.
    Volume int64     `json:"v,omitempty"`      // Volume represents the trading volume during the candle's time.
    VWAP   float64   `json:"vwap,omitempty"`   // VWAP is the Volume Weighted Average Price, optional.
    N      int64     `json:"n,omitempty"`      // N is the number of trades that occurred, optional.
}
```

Candle represents a single candle in a stock candlestick chart, encapsulating the time, open, high, low, close prices, volume, and optionally the symbol, VWAP, and number of trades.

#### Generated By

- <a href="#StockCandlesResponse.Unpack">`StockCandlesResponse.Unpack()`</a>

  Generates Candle instances from a StockCandlesResponse.

- <a href="#BulkStockCandlesResponse.Unpack">`BulkStockCandlesResponse.Unpack()`</a>

  Generates Candle instances from a BulkStockStockCandlesResponse.

- <a href="#IndicesCandlesResponse.Unpack">`IndicesCandlesResponse.Unpack()`</a>

  Generates Candle instances from a IndicesCandlesResponse.


#### Methods

- <a href="#Candle.String">`String() string`</a>

  Provides a string representation of the Candle.

- <a href="#Candle.Equals">`Equals(other Candle) bool`</a>

  Checks if two Candle instances are equal.

- <a href="#Candle.MarshalJSON">`MarshalJSON() ([]byte, error)`</a>

  Customizes the JSON output of Candle.

- <a href="#Candle.UnmarshalJSON">`UnmarshalJSON(data []byte) error`</a>

  Customizes the JSON input processing of Candle.


### Notes

- The VWAP, N fields are optional and will only be present in v2 Stock Candles.
- The Volume field is optional and will not be present in Index Candles.
- The Symbol field is optional and only be present in candles that were generated using the bulkcandles endpoint.



<a name="Candle.Clone"></a>
### func \(Candle\) Clone

```go
func (c Candle) Clone() Candle
```

Clones the current Candle instance, creating a new instance with the same values. This method is useful when you need a copy of a Candle instance without modifying the original instance.

#### Returns

- `Candle`

  A new Candle instance with the same values as the current instance.


<a name="Candle.Equals"></a>
### func \(Candle\) Equals

```go
func (c Candle) Equals(other Candle) bool
```

Equals compares the current Candle instance with another Candle instance to determine if they represent the same candle data. This method is useful for validating if two Candle instances have identical properties, including symbol, date/time, open, high, low, close prices, volume, VWAP, and number of trades. It's primarily used in scenarios where candle data integrity needs to be verified or when deduplicating candle data.

#### Parameters

- `Candle`

  The other Candle instance to compare against the current instance.


#### Returns

- `bool`

  Indicates whether the two Candle instances are identical. True if all properties match, false otherwise.


#### Notes

- This method performs a deep equality check on all Candle properties, including date/time which is compared using the Equal method from the time package to account for potential timezone differences.

<a name="Candle.IsAfter"></a>
### func \(Candle\) IsAfter

```go
func (c Candle) IsAfter(other Candle) bool
```

IsAfter determines if the current Candle instance occurred after another specified Candle instance. This method is useful for chronological comparisons between two Candle instances, particularly in time series analysis or when organizing historical financial data in ascending order.

#### Parameters

- `other Candle`

  The Candle instance to compare with the current Candle instance.


#### Returns

- `bool`

  Indicates whether the current Candle's date is after the 'other' Candle's date. Returns true if it is; otherwise, false.


<a name="Candle.IsBefore"></a>
### func \(Candle\) IsBefore

```go
func (c Candle) IsBefore(other Candle) bool
```

IsBefore determines whether the current Candle instance occurred before another specified Candle instance. This method is primarily used for comparing the dates of two Candle instances to establish their chronological order, which can be useful in time series analysis or when organizing historical financial data.

#### Parameters

- `other Candle`

  The Candle instance to compare with the current Candle instance.


#### Returns

- `bool`

  Returns true if the date of the current Candle instance is before the date of the 'other' Candle instance; otherwise, returns false.


#### Notes

- This method only compares the dates of the Candle instances, ignoring other fields such as Open, Close, High, Low, etc.

<a name="Candle.IsValid"></a>
### func \(Candle\) IsValid

```go
func (c Candle) IsValid() bool
```

IsValid evaluates the financial data of a Candle to determine its validity. This method is essential for ensuring that the Candle's data adheres to basic financial integrity rules, making it a critical step before performing further analysis or operations with Candle data. A Candle is deemed valid if its high, open, and close prices are logically consistent with each other and its volume is non\-negative.

#### Returns

- `bool`

  Indicates whether the Candle is valid based on its financial data. Returns true if all validity criteria are met; otherwise, false.


<a name="Candle.MarshalJSON"></a>
### func \(Candle\) MarshalJSON

```go
func (c Candle) MarshalJSON() ([]byte, error)
```

MarshalJSON customizes the JSON output of the Candle struct, primarily used for converting the Candle data into a JSON format that includes the Date as a Unix timestamp instead of the standard time.Time format. This method is particularly useful when the Candle data needs to be serialized into JSON for storage or transmission over networks where a compact and universally understood date format is preferred.

#### Returns

- `[]byte`

  The JSON\-encoded representation of the Candle.

- `error`

  An error if the JSON marshaling fails.


#### Notes

- The Date field of the Candle is converted to a Unix timestamp to facilitate easier handling of date and time in JSON.

<a name="Candle.String"></a>
### func \(Candle\) String

```go
func (c Candle) String() string
```

String provides a textual representation of the Candle instance. This method is primarily used for logging or debugging purposes, where a developer needs a quick and easy way to view the contents of a Candle instance in a human\-readable format.

#### Returns

- `string`

  A string that represents the Candle instance, including its symbol, date/time, open, high, low, close prices, volume, VWAP, and number of trades, if available.


#### Notes

- The output format is designed to be easily readable, with each field labeled and separated by commas.
- Fields that are not applicable or not set \(e.g., VWAP, N, Volume for Index Candles\) are omitted from the output.

<a name="Candle.UnmarshalJSON"></a>
### func \(\*Candle\) UnmarshalJSON

```go
func (c *Candle) UnmarshalJSON(data []byte) error
```

UnmarshalJSON customizes the JSON input processing of Candle.

UnmarshalJSON customizes the JSON input processing for the Candle struct, allowing for the Date field to be correctly interpreted from a Unix timestamp \(integer\) back into a Go time.Time object. This method is essential for deserializing Candle data received in JSON format, where date and time are represented as Unix timestamps, ensuring the Candle struct accurately reflects the original data.

#### Parameters

- `data []byte`

  The JSON\-encoded data that is to be unmarshaled into the Candle struct.


#### Returns

- `error`

  An error if the JSON unmarshaling fails, nil otherwise.


#### Notes

- The Date field in the JSON is expected to be a Unix timestamp \(integer\). This method converts it back to a time.Time object, ensuring the Candle struct's Date field is correctly populated.

<a name="IndicesCandlesResponse"></a>
## type IndicesCandlesResponse

```go
type IndicesCandlesResponse struct {
    Date  []int64   `json:"t"` // Date holds the Unix timestamps for each candle, representing the time at which each candle was opened.
    Open  []float64 `json:"o"` // Open contains the opening prices for each candle in the response.
    High  []float64 `json:"h"` // High includes the highest prices reached during the time period each candle represents.
    Low   []float64 `json:"l"` // Low encompasses the lowest prices during the candle's time period.
    Close []float64 `json:"c"` // Close contains the closing prices for each candle, marking the final price at the end of each candle's time period.
}
```

IndicesCandlesResponse represents the response structure for indices candles data. It includes slices for time, open, high, low, and close values of the indices.

#### Generated By

- <a href="#IndexCandlesRequest.Packed">`IndexCandlesRequest.Packed()`</a>

  This method sends the IndicesCandlesRequest to the Market Data API and returns the IndicesCandlesResponse. It handles the actual communication with the \[/v1/indices/candles/] endpoint, sending the request, and returns a packed response that strictly conforms to the Market Data JSON response without unpacking the result into individual candle structs.


#### Methods

- <a href="#IndicesCandlesResponse.String">`String()`</a>

  Provides a formatted string representation of the IndicesCandlesResponse instance. This method is primarily used for logging or debugging purposes, allowing the user to easily view the contents of an IndicesCandlesResponse object in a human\-readable format. It concatenates the time, open, high, low, and close values of the indices into a single string.

- <a href="#IndicesCandlesResponse.Unpack">`Unpack() ([]Candle, error)`</a>

  Unpacks the IndicesCandlesResponse into a slice of Candle structs, checking for errors in data consistency.

- <a href="#IndicesCandlesResponse.MarshalJSON">`MarshalJSON()`</a>

  Marshals the IndicesCandlesResponse into a JSON object with ordered keys.

- <a href="#IndicesCandlesResponse.UnmarshalJSON">`UnmarshalJSON(data []byte)`</a>

  Custom unmarshals a JSON object into the IndicesCandlesResponse, including validation.

- <a href="#IndicesCandlesResponse.Validate">`Validate()`</a>

  Runs checks for time in ascending order, equal slice lengths, and no empty slices.

- <a href="#IndicesCandlesResponse.IsValid">`IsValid()`</a>

  Checks if the IndicesCandlesResponse passes all validation checks and returns a boolean.




<a name="IndicesCandlesResponse.IsValid"></a>
### func \(\*IndicesCandlesResponse\) IsValid

```go
func (icr *IndicesCandlesResponse) IsValid() bool
```

#### Returns

- `bool`

  Indicates whether the IndicesCandlesResponse is valid.


<a name="IndicesCandlesResponse.MarshalJSON"></a>
### func \(\*IndicesCandlesResponse\) MarshalJSON

```go
func (icr *IndicesCandlesResponse) MarshalJSON() ([]byte, error)
```

MarshalJSON marshals the IndicesCandlesResponse struct into a JSON object, ensuring the keys are ordered as specified. This method is particularly useful when a consistent JSON structure with ordered keys is required for external interfaces or storage. The "s" key is set to "ok" to indicate successful marshaling, followed by the indices data keys "t", "o", "h", "l", and "c".

#### Returns

- `[]byte`

  A byte slice representing the marshaled JSON object. The keys within the JSON object are ordered as "s", "t", "o", "h", "l", and "c".

- `error`

  An error object if marshaling fails, otherwise nil.


<a name="IndicesCandlesResponse.String"></a>
### func \(\*IndicesCandlesResponse\) String

```go
func (icr *IndicesCandlesResponse) String() string
```

String provides a formatted string representation of the IndicesCandlesResponse instance. This method is primarily used for logging or debugging purposes, allowing the user to easily view the contents of an IndicesCandlesResponse object in a human\-readable format. It concatenates the time, open, high, low, and close values of the indices into a single string.

#### Returns

- `string`

  A formatted string containing the time, open, high, low, and close values of the indices.


<a name="IndicesCandlesResponse.UnmarshalJSON"></a>
### func \(\*IndicesCandlesResponse\) UnmarshalJSON

```go
func (icr *IndicesCandlesResponse) UnmarshalJSON(data []byte) error
```

UnmarshalJSON custom unmarshals a JSON object into the IndicesCandlesResponse, incorporating validation to ensure the data integrity of the unmarshaled object. This method is essential for converting JSON data into a structured IndicesCandlesResponse object while ensuring that the data adheres to expected formats and constraints.

#### Parameters

- `[]byte`

  A byte slice of the JSON object to be unmarshaled.


#### Returns

- `error`

  An error if unmarshaling or validation fails, otherwise nil.


<a name="IndicesCandlesResponse.Unpack"></a>
### func \(\*IndicesCandlesResponse\) Unpack

```go
func (icr *IndicesCandlesResponse) Unpack() ([]Candle, error)
```

Unpack converts the IndicesCandlesResponse into a slice of IndexCandle.

#### Returns

- `[]Candle`

  A slice of [Candle](<#Candle>) that holds the OHLC data.

- `error`

  An error object that indicates a failure in unpacking the response.


<a name="IndicesCandlesResponse.Validate"></a>
### func \(\*IndicesCandlesResponse\) Validate

```go
func (icr *IndicesCandlesResponse) Validate() error
```

Validate performs multiple checks on the IndicesCandlesResponse to ensure data integrity. This method is crucial for verifying that the response data is consistent and reliable, specifically checking for time sequence, equal length of data slices, and the presence of data in each slice. It's used to preemptively catch and handle data\-related errors before they can affect downstream processes.

#### Returns

- `error`

  An error if any validation check fails, otherwise nil. This allows for easy identification of data integrity issues.



