package models

import (
	"fmt"
)

// OptionLookupResponse encapsulates the response data for an option lookup request, primarily containing the option's symbol.
//
// # Generated By
//
//   - OptionLookupRequest.Packed(): Unmarshals the JSON response from OptionLookupRequest into OptionLookupResponse.
//
// # Methods
//
//   - IsValid() bool: Checks if the OptionLookupResponse is valid by verifying the OptionSymbol is not empty.
//   - String() string: Provides a string representation of the OptionLookupResponse, including the OptionSymbol.
//   - Unpack() (string, error): Validates the OptionLookupResponse and returns the OptionSymbol if valid; otherwise, returns an error.
//
// # Notes
//
//   - This struct is primarily used for handling the response of an options lookup request in financial market data applications.
type OptionLookupResponse struct {
	OptionSymbol string `json:"optionSymbol"` // OptionSymbol is the symbol of the option.
}

// IsValid determines the validity of the OptionLookupResponse. It is primarily used to ensure that the response received from an option lookup request contains a non-empty OptionSymbol, indicating a successful lookup and a valid option.
//
// # Returns
//
//   - bool: Indicates the validity of the OptionLookupResponse. Returns true if the OptionSymbol is not empty, otherwise false.
func (olr *OptionLookupResponse) IsValid() bool {
	return olr.OptionSymbol != ""
}

// String provides a human-readable representation of the OptionLookupResponse, including the OptionSymbol. This method is useful for logging, debugging, or displaying the response in a format that is easy to read and understand.

// # Returns
//   - string: A formatted string that encapsulates the OptionLookupResponse details, particularly the OptionSymbol.

// # Notes
//   - This method is primarily intended for debugging purposes or when there's a need to log the response details in a human-readable format.
func (olr *OptionLookupResponse) String() string {
	return fmt.Sprintf("OptionLookupResponse{OptionSymbol: %q}", olr.OptionSymbol)
}

// Unpack checks the validity of the OptionLookupResponse and returns the OptionSymbol if the response is deemed valid. This method is primarily used when one needs to extract the OptionSymbol from a valid OptionLookupResponse, ensuring that the response is not empty or malformed before proceeding with further processing.
//
// # Returns
//
//   - string: The OptionSymbol contained within a valid OptionLookupResponse.
//   - error: An error indicating that the OptionLookupResponse is invalid, typically due to an empty OptionSymbol.
//
// # Notes
//
//   - This method is crucial for error handling and data validation in financial market data applications, ensuring that only valid responses are processed.
func (olr *OptionLookupResponse) Unpack() (string, error) {
	if !olr.IsValid() {
		return "", fmt.Errorf("invalid OptionLookupResponse")
	}
	return olr.OptionSymbol, nil
}
