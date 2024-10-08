// Package client provides functionalities to interact with the Options Chain endpoint.
// Retrieve a a complete or filtered options chain for a given underlying symbol. Both real-time and historical requests are possible.
//
// # Making Requests
//
// Utilize [OptionChainRequest] for querying the endpoint through one of the three available methods:
//
//	| Method     | Execution Level | Return Type                  | Description                                                                                                                      |
//	|------------|-----------------|------------------------------|----------------------------------------------------------------------------------------------------------------------------------|
//	| **Get**    | Direct          | `[]OptionQuote`              | Immediately fetches a slice of `[]OptionQuote`, allowing direct access to the options chain data.                                |
//	| **Packed** | Intermediate    | `*OptionQuotesResponse`      | Delivers a `*OptionQuotesResponse` object containing the data, which requires unpacking to access the `OptionQuote` data.        |
//	| **Raw**    | Low-level       | `*resty.Response`            | Offers the unprocessed `*resty.Response` for those seeking full control and access to the raw JSON or `*http.Response`.          |
package client

import (
	"context"
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
	"github.com/go-resty/resty/v2"
)

// OptionChainRequest represents a request to the [/v1/options/chain/] endpoint.
// It encapsulates parameters for symbol, date, and various option-specific parameters to be used in the request.
// This struct provides methods to set these parameters, such as UnderlyingSymbol(), Date(), Expiration(), and Strike(), among others.
//
// # Generated By
//
//   - OptionChain() *OptionChainRequest: OptionChain creates a new *OptionChainRequest and returns a pointer to the request allowing for method chaining.
//
// # Setter Methods
//
// These methods are used to set the parameters of the request. They allow for method chaining
// by returning a pointer to the *OptionChainRequest instance they modify.
//
//   - UnderlyingSymbol(string) *OptionChainRequest: Sets the underlying symbol parameter for the request.
//   - Date(interface{}) *OptionChainRequest: Sets the date parameter for the request.
//   - Expiration(interface{}) *OptionChainRequest: Sets the expiration parameter for the request.
//   - Strike(float64) *OptionChainRequest: Sets the strike price parameter for the request.
//   - Monthly(bool) *OptionChainRequest: Inclues or excludes monthly option expirations from the result.
//   - Weekly(bool) *OptionChainRequest: Inclues or excludes weekly option expirations from the result.
//   - Quarterly(bool) *OptionChainRequest: Inclues or excludes quarterly option expirations from the result.
//   - Nonstandard(bool) *OptionChainRequest: Inclues or excludes non-standard option expirations from the result.
//   - Month(int) *OptionChainRequest: Requests results from the specific specific month.
//   - Year(int) *OptionChainRequest: Sets the year parameter for the request.
//   - DTE(int) *OptionChainRequest: Sets the Days to Expiration parameter for the request.
//   - Delta(float64) *OptionChainRequest: Sets the Delta parameter for the request.
//   - Side(string) *OptionChainRequest: Sets the side of the market (call or put) for the request.
//   - Range(string) *OptionChainRequest: Sets the range parameter for the request.
//   - StrikeLimit(int) *OptionChainRequest: Sets the maximum number of strike prices to be included in the option chain.
//   - MinOpenInterest(int) *OptionChainRequest: Sets the minimum open interest for options to be included in the option chain.
//   - MinVolume(int) *OptionChainRequest: Sets the minimum volume for options to be included in the option chain.
//   - MaxBidAskSpread(float64) *OptionChainRequest: Sets the maximum bid-ask spread for options to be included in the option chain.
//   - MaxBidAskSpreadPct(float64) *OptionChainRequest: Sets the maximum bid-ask spread percentage for options to be included in the option chain.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Get() ([]OptionQuote, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//   - Packed() (*OptionQuotesResponse, error): Packs the request parameters and sends the request, returning a *OptionQuotesResponse response.
//   - Raw() (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//
// [/v1/options/chain/]: https://www.marketdata.app/docs/api/options/chain
type OptionChainRequest struct {
	*baseRequest
	symbolParams *parameters.SymbolParams
	dateParams   *parameters.DateParams
	optionParams *parameters.OptionParams
}

// UnderlyingSymbol sets the underlyingSymbol parameter for the OptionChainRequest.
// This method is used to specify the symbol of the underlying asset for which the option chain is requested.
// It modifies the symbolParams field of the OptionChainRequest instance to store the symbol value.
//
// # Parameters
//
//   - string: The underlying symbol to be set.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining, where multiple setter methods can be called in a single statement. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) UnderlyingSymbol(q string) *OptionChainRequest {
	if ocr == nil {
		return nil
	}
	err := ocr.symbolParams.SetSymbol(q)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Date sets the date parameter for the OptionChainRequest.
// This method is used to specify the date for which the option chain is requested.
// It modifies the dateParams field of the OptionChainRequest instance to store the date value.
//
// # Parameters
//
//   - interface{}: An interface{} that represents the starting date. It can be a string, a time.Time object, a Unix timestamp or any other type that the underlying dates package can process.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Date(q interface{}) *OptionChainRequest {
	if ocr.dateParams == nil {
		ocr.dateParams = &parameters.DateParams{}
	}
	err := ocr.dateParams.SetDate(q)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Expiration sets the expiration parameter for the OptionChainRequest.
// This method is used to specify the expiration date for the options in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the expiration value.
//
// # Parameters
//
//   - interface{}: An interface{} representing the expiration date to be set.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Expiration(q interface{}) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetExpiration(q)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Month sets the month parameter for the OptionChainRequest.
// This method is used to specify the month for which the option chain is requested.
// It modifies the optionParams field of the OptionChainRequest instance to store the month value.
//
// # Parameters
//
//   - int: The month to be set.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Month(month int) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetMonth(month)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Year sets the year parameter for the OptionChainRequest.
// This method is used to specify the year for which the option chain is requested.
// It modifies the optionParams field of the OptionChainRequest instance to store the year value.
//
// # Parameters
//
//   - int: The year to be set.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Year(year int) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetYear(year)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Weekly sets the weekly parameter for the OptionChainRequest.
// This method is used to specify whether to include weekly options in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the weekly value.
//
// # Parameters
//
//   - bool: Whether to include weekly options.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Weekly(weekly bool) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetWeekly(weekly)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Monthly sets the monthly parameter for the OptionChainRequest.
// This method is used to specify whether to include monthly options in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the monthly value.
//
// # Parameters
//
//   - bool: Whether to include monthly options.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Monthly(monthly bool) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetMonthly(monthly)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Quarterly sets the quarterly parameter for the OptionChainRequest.
// This method is used to specify whether to include quarterly options in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the quarterly value.
//
// # Parameters
//
//   - bool: Whether to include quarterly options.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Quarterly(quarterly bool) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetQuarterly(quarterly)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Nonstandard sets the nonstandard parameter for the OptionChainRequest.
// This method is used to specify whether to include nonstandard options in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the nonstandard value.
//
// # Parameters
//
//   - bool: Whether to include nonstandard options.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Nonstandard(nonstandard bool) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetNonstandard(nonstandard)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// DTE (Days to Expiration) sets the DTE parameter for the OptionChainRequest.
// This method specifies the number of days to expiration for the options in the option chain.
//
// # Parameters
//
//   - int: Requests an expiration date from the option chain based on the number of days from the present date.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) DTE(dte int) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetDTE(dte)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Delta sets the Delta parameter for the OptionChainRequest.
// This method is used to specify a particular Delta value for the options in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the Delta value.
//
// # Parameters
//
//   - float64: The Delta value to be set.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Delta(delta float64) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetDelta(delta)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Side sets the Side parameter for the OptionChainRequest.
// This method is used to specify the side of the market for the options in the option chain (e.g., call or put).
// It modifies the optionParams field of the OptionChainRequest instance to store the side value.
//
// # Parameters
//
//   - string: A string representing the side of the market chain to be set. Expected value is "call" or "put".
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Side(side string) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetSide(side)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Range sets the Range parameter for the OptionChainRequest.
// This method is used to specify the range of options to be included in the option chain based on their strike price.
// It modifies the optionParams field of the OptionChainRequest instance to store the range value.
//
// # Parameters
//
//   - string: A string representing the range of options to be included. The expected values can vary, such as "ITM" (In The Money), "OTM" (Out of The Money), "ATM" (At The Money), etc.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Range(rangeParam string) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetRange(rangeParam)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// Strike sets the strike price parameter for the OptionChainRequest.
// This method is used to specify a particular strike price for the options in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the strike price value.
//
// # Parameters
//
//   - float64: The strike price to be set.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) Strike(strike float64) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetStrike(strike)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// StrikeLimit sets the StrikeLimit parameter for the OptionChainRequest.
// This method is used to specify the maximum number of strike prices to be included in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the strike limit value.
//
// # Parameters
//
//   - int: The maximum number of strikes to be included.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) StrikeLimit(strikeLimit int) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetStrikeLimit(strikeLimit)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// MinOpenInterest sets the MinOpenInterest parameter for the OptionChainRequest.
// This method is used to specify the minimum open interest for options to be included in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the minimum open interest value.
//
// # Parameters
//
//   - int: An int representing the minimum open interest required to be included in the result.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) MinOpenInterest(minOpenInterest int) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetMinOpenInterest(minOpenInterest)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// MinVolume sets the MinVolume parameter for the OptionChainRequest.
// This method is used to specify the minimum volume for options to be included in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the minimum volume value.
//
// # Parameters
//
//   - int: An int representing the minimum volume needed to be included in the result.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) MinVolume(minVolume int) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetMinVolume(minVolume)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// MaxBidAskSpread sets the MaxBidAskSpread parameter for the OptionChainRequest.
// This method is used to specify the maximum bid-ask spread for options to be included in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the maximum bid-ask spread value.
//
// # Parameters
//
//   - float64: A float64 representing the maximum bid-ask spread neeeded to be included in the result.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) MaxBidAskSpread(maxBidAskSpread float64) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetMaxBidAskSpread(maxBidAskSpread)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// MaxBidAskSpreadPct sets the MaxBidAskSpreadPct parameter for the OptionChainRequest.
// This method is used to specify the maximum bid-ask spread percentage for options to be included in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the maximum bid-ask spread percentage value.
//
// # Parameters
//
//   - float64: A float64 representing the maximum bid-ask spread percentage to be included in the result.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) MaxBidAskSpreadPct(maxBidAskSpreadPct float64) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetMaxBidAskSpreadPct(maxBidAskSpreadPct)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// MinAsk sets the MinAsk parameter for the OptionChainRequest.
// This method is used to specify the minimum ask price for options to be included in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the minimum ask price value.
//
// # Parameters
//
//   - float64: A float64 representing the minimum ask price to be included in the result.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) MinAsk(minAsk float64) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetMinAsk(minAsk)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// MaxAsk sets the MaxAsk parameter for the OptionChainRequest.
// This method is used to specify the maximum ask price for options to be included in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the maximum ask price value.
//
// # Parameters
//
//   - float64: A float64 representing the maximum ask price to be included in the result.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) MaxAsk(maxAsk float64) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetMaxAsk(maxAsk)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// MinBid sets the MinBid parameter for the OptionChainRequest.
// This method is used to specify the minimum bid price for options to be included in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the minimum bid price value.
//
// # Parameters
//
//   - float64: A float64 representing the minimum bid price to be included in the result.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) MinBid(minBid float64) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetMinBid(minBid)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// MaxBid sets the MaxBid parameter for the OptionChainRequest.
// This method is used to specify the maximum bid price for options to be included in the option chain.
// It modifies the optionParams field of the OptionChainRequest instance to store the maximum bid price value.
//
// # Parameters
//
//   - float64: A float64 representing the maximum bid price to be included in the result.
//
// # Returns
//
//   - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
func (ocr *OptionChainRequest) MaxBid(maxBid float64) *OptionChainRequest {
	if ocr.optionParams == nil {
		ocr.optionParams = &parameters.OptionParams{}
	}
	err := ocr.optionParams.SetMaxBid(maxBid)
	if err != nil {
		ocr.Error = err
	}
	return ocr
}

// getParams packs the OptionChainRequest struct into a slice of interface{} and returns it.
// This method is used to gather all the parameters set in the OptionChainRequest into a single slice
// for easier manipulation and usage in subsequent requests.
//
// # Returns
//
//   - []parameters.MarketDataParam: A slice containing all the parameters set in the OptionChainRequest.
//   - error: An error object indicating failure to pack the parameters, nil if successful.
func (ocr *OptionChainRequest) getParams() ([]parameters.MarketDataParam, error) {
	if ocr == nil {
		return nil, fmt.Errorf("OptionChainRequest is nil")
	}
	params := []parameters.MarketDataParam{ocr.symbolParams, ocr.dateParams, ocr.optionParams}
	return params, nil
}

// Raw executes the OptionChainRequest with the provided context and returns the raw *resty.Response.
// This method uses the default client for this request. The *resty.Response can be directly used to access the raw JSON or *http.Response.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - *resty.Response: The raw HTTP response from the executed OptionChainRequest.
//   - error: An error object if the OptionChainRequest is nil or if an error occurs during the request execution.
func (ocr *OptionChainRequest) Raw(ctx context.Context) (*resty.Response, error) {
	return ocr.baseRequest.Raw(ctx)
}

// Packed sends the OptionChainRequest with the provided context and returns the OptionChainResponse.
// This method checks if the OptionChainRequest receiver is nil, returning an error if true.
// It proceeds to send the request using the default client and returns the OptionChainResponse along with any error encountered during the request.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - *models.OptionQuotesResponse: A pointer to the OptionQuotesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (ocr *OptionChainRequest) Packed(ctx context.Context) (*models.OptionQuotesResponse, error) {
	if ocr == nil {
		return nil, fmt.Errorf("OptionChainRequest is nil")
	}

	var ocrResp models.OptionQuotesResponse
	_, err := ocr.baseRequest.client.getFromRequest(ctx, ocr.baseRequest, &ocrResp)
	if err != nil {
		return nil, err
	}

	return &ocrResp, nil
}

// Get sends the OptionChainRequest with the provided context, unpacks the OptionChainResponse, and returns a slice of OptionQuote.
// It returns an error if the request or unpacking fails. This method is crucial for obtaining the actual option chain data
// from the option chain request. The method first checks if the OptionChainRequest receiver is nil, which would
// result in an error as the request cannot be sent. It then proceeds to send the request using the default client.
// Upon receiving the response, it unpacks the data into a slice of OptionQuote using the Unpack method from the response.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - []models.OptionQuote: A slice of OptionQuote containing the unpacked option chain data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (ocr *OptionChainRequest) Get(ctx context.Context) ([]models.OptionQuote, error) {
	if ocr == nil {
		return nil, fmt.Errorf("OptionChainRequest is nil")
	}

	// Use the Packed method to make the request
	ocrResp, err := ocr.Packed(ctx)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := ocrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// OptionChain creates a new OptionChainRequest and associates it with the default client. This function initializes the request
// with default parameters for symbol, date, and option-specific parameters, and sets the request path based on
// the predefined endpoints for option chains.
//
// # Returns
//
//   - *OptionChainRequest: A pointer to the newly created OptionChainRequest with default parameters and associated client.
func OptionChain() *OptionChainRequest {
	baseReq := newBaseRequest()
	baseReq.path = endpoints[1]["options"]["chain"]

	ocr := &OptionChainRequest{
		baseRequest:  baseReq,
		symbolParams: &parameters.SymbolParams{},
		optionParams: &parameters.OptionParams{},
		dateParams:   &parameters.DateParams{},
	}

	baseReq.child = ocr

	return ocr
}
