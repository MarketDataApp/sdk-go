# Adding New Endpoints

To add a new endpoint to the Market Data Go SDK, follow these steps:

### 1. **Update Endpoints File**: 

Add the new endpoint to the endpoints map in endpoints.go. Specify the version and category it belongs to.

```go
// Example for adding a new "futures" quote in version 1
1: {
    "futures": {
        "quotes": "/v1/futures/quotes/{symbol}/",
    },
```

### 2. **Update baseRequest.go**: 

Add a type assertion for the new request type.

```go
// Example for adding a new FuturesQuoteRequest in version 1
	if fqr, ok := br.child.(*FuturesQuoteRequest); ok {
		params, err := fqr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}
```

### 3. **Create Request Struct/Methods**: 
  - Define a struct in the `client` package to represent the new request. Use the existing requests as a guide. 
  - Attach the existing parameters, or design new parameters in the `helpers/parameters` package. Most common parameters are already defined.
  - Create setter methods for the request struct that passes throught the parameters to the `helpers/parameters` that you've imported to use. These methods return a pointer to the struct and store errors, allowing the user to use the builder pattern to define the parameters.
  - Create a function to initialize a new empty request that attaches the default client or a user-provided client.

```go
package client

import (
    "github.com/MarketDataApp/sdk-go/helpers/parameters"
)

// FuturesQuoteRequest is the struct that sets/stores the request parameters.
type FuturesQuoteRequest struct {
	*baseRequest
	symbolParams       *parameters.SymbolParams
}

// Symbol sets the FuturesQuoteRequest ticket symbol.
func (fqr *FuturesQuoteRequest) Symbol(q string) *FuturesQuoteRequest {
	if fqr == nil {
		return nil
	}
	err := fqr.symbolParams.SetSymbol(q)
	if err != nil {
		fqr.Error = err
	}
	return fqr
}

// FuturesQuote initializes a new FuturesQuoteRequest.
func FuturesQuote(client ...*MarketDataClient) *FuturesQuoteRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["futures"]["quotes"]

	fqr := &StockQuoteRequest{
		baseRequest:        baseReq,
		symbolParams:       &parameters.SymbolParams{},
	}

	baseReq.child = fqr

	return fqr
}
```

### 4. **Create Response Struct**: 

Define a struct in the `models` package to unmarshal the JSON response from the Market Data API. This struct should mirror the JSON structure returned by the new endpoint. Market Data structs are typically JSON arrays.

```go
package models

// FuturesQuoteResponse represents the JSON response structure for futures quotes.
type FuturesQuoteResponse struct {
    Symbol []string  `json:"symbol"`
    Bid    []float64 `json:"bid"`
    Ask    []float64 `json:"ask"`
    Time   []int64   `json:"time"`
}
```

### 5. **Create Struct for Unpacked Object**: 

Define a struct that represents the unpacked object. This struct is what the Unpack method will return.

```go
// FuturesQuote represents a single futures quote.
type FuturesQuote struct {
    Symbol string
    Bid    float64
    Ask    float64
    Time   time.Time
}
```

### 6. **Create Unpack Method**: 

Implement an Unpack method for your response struct to convert the packed JSON array format into individual struct objects. This method should be part of the response struct in the `models` package.

```go
// Unpack converts packed FuturesQuoteResponse into a slice of individual structs.
func (fpr *FuturesQuoteResponse) Unpack() ([]FuturesQuote, error) {
    var quotes []FuturesQuote
    for i := range fpr.Symbol {
        quote := FuturesQuote{
            Symbol: fpr.Symbol[i],
            Bid:    fpr.Bid[i],
            Ask:    fpr.Ask[i],
            Time:   time.Unix(fpr.Time[i], 0),
        }
        quotes = append(quotes, quote)
    }
    return quotes, nil
```

### 7. **Implement Packed and Get Methods**: 

In the client package, create methods for your endpoint that return the packed response (Packed) and the unpacked response (Get). These methods should utilize the base request functionality and your response struct.

By following these steps, you can extend the Market Data Go SDK to support new endpoints, ensuring that the SDK remains a comprehensive tool for accessing financial data.

### 8. **Create Documentation, Tests and Testable Examples**: 

Document with DocStrings, create tests, and create testable examples for both the `Get` and `Packed` methods.