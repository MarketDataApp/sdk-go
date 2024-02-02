package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// OptionChainRequest represents a request to the /options/chain endpoint.
// It encapsulates parameters for symbol, date, and various option-specific parameters to be used in the request.
// This struct provides methods to set these parameters, such as UnderlyingSymbol(), Date(), Expiration(), and Strike(), among others.
//
// Public Methods:
// - UnderlyingSymbol(q string) *OptionChainRequest: Sets the underlying symbol parameter for the request.
// - Date(q interface{}) *OptionChainRequest: Sets the date parameter for the request.
// - Expiration(q interface{}) *OptionChainRequest: Sets the expiration parameter for the request.
// - Strike(strike float64) *OptionChainRequest: Sets the strike price parameter for the request.
// - and other option-specific parameter setting methods like Month(), Year(), Weekly(), Monthly(), etc.
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
// Parameters:
// - q: A string representing the underlying symbol to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining, where multiple setter methods can be called in a single statement. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - q: An interface{} representing the date to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - q: An interface{} representing the expiration date to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - month: An int representing the month to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - year: An int representing the year to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - weekly: A bool indicating whether to include weekly options.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - monthly: A bool indicating whether to include monthly options.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - quarterly: A bool indicating whether to include quarterly options.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - nonstandard: A bool indicating whether to include nonstandard options.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// It modifies the optionParams field of the OptionChainRequest instance to store the DTE value.
//
// Parameters:
// - dte: An int representing the days to expiration to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - delta: A float64 representing the Delta value to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - side: A string representing the side of the market to be set. Expected values are typically "call" or "put".
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - rangeParam: A string representing the range of options to be included. The expected values can vary, such as "ITM" (In The Money), "OTM" (Out of The Money), "ATM" (At The Money), etc.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - strike: A float64 representing the strike price to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - strikeLimit: An int representing the maximum number of strike prices to be included.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - minOpenInterest: An int representing the minimum open interest to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - minVolume: An int representing the minimum volume to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - maxBidAskSpread: A float64 representing the maximum bid-ask spread to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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
// Parameters:
// - maxBidAskSpreadPct: A float64 representing the maximum bid-ask spread percentage to be set.
//
// Returns:
// - *OptionChainRequest: This method returns a pointer to the OptionChainRequest instance it was called on. This allows for method chaining. If the receiver (*OptionChainRequest) is nil, it returns nil to prevent a panic.
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

// getParams packs the OptionChainRequest struct into a slice of interface{} and returns it.
// This method is used to gather all the parameters set in the OptionChainRequest into a single slice
// for easier manipulation and usage in subsequent requests.
//
// Returns:
// - []parameters.MarketDataParam: A slice containing all the parameters set in the OptionChainRequest.
// - error: An error object indicating failure to pack the parameters, nil if successful.
func (ocr *OptionChainRequest) getParams() ([]parameters.MarketDataParam, error) {
	if ocr == nil {
		return nil, fmt.Errorf("OptionChainRequest is nil")
	}
	params := []parameters.MarketDataParam{ocr.symbolParams, ocr.dateParams, ocr.optionParams}
	return params, nil
}

// Packed sends the OptionChainRequest and returns the OptionChainResponse.
// This method checks if the OptionChainRequest receiver is nil, returning an error if true.
// Otherwise, it proceeds to send the request and returns the OptionChainResponse along with any error encountered during the request.
//
// Returns:
// - *models.OptionQuotesResponse: A pointer to the OptionQuotesResponse obtained from the request.
// - error: An error object that indicates a failure in sending the request.
func (ocr *OptionChainRequest) Packed() (*models.OptionQuotesResponse, error) {
	if ocr == nil {
		return nil, fmt.Errorf("OptionChainRequest is nil")
	}
	var ocrResp models.OptionQuotesResponse
	_, err := ocr.baseRequest.client.GetFromRequest(ocr.baseRequest, &ocrResp)
	if err != nil {
		return nil, err
	}

	return &ocrResp, nil
}

// Get sends the OptionChainRequest, unpacks the OptionChainResponse, and returns a slice of OptionQuote.
// It returns an error if the request or unpacking fails. This method is crucial for obtaining the actual option chain data
// from the option chain request. The method first checks if the OptionChainRequest receiver is nil, which would
// result in an error as the request cannot be sent. It then proceeds to send the request using the Packed method.
// Upon receiving the response, it unpacks the data into a slice of OptionQuote using the Unpack method from the response.
//
// Returns:
// - []models.OptionQuote: A slice of OptionQuote containing the unpacked option chain data from the response.
// - error: An error object that indicates a failure in sending the request or unpacking the response.
func (ocr *OptionChainRequest) Get() ([]models.OptionQuote, error) {
	if ocr == nil {
		return nil, fmt.Errorf("OptionChainRequest is nil")
	}

	// Use the Packed method to make the request
	ocrResp, err := ocr.Packed()
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

// OptionChain creates a new OptionChainRequest and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for symbol, date, and option-specific parameters, and sets the request path based on
// the predefined endpoints for option chains.
// Parameters:
// - clients: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided,
//   the default client is used.
// Returns:
// - *OptionChainRequest: A pointer to the newly created OptionChainRequest with default parameters and associated client.
func OptionChain(client ...*MarketDataClient) *OptionChainRequest {
	baseReq := newBaseRequest(client...)
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
