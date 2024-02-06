package models

import (
	"fmt"
)

// OptionLookupResponse represents the response structure for an option lookup request.
// It contains the symbol of the option being looked up.
type OptionLookupResponse struct {
	OptionSymbol string `json:"optionSymbol"` // OptionSymbol is the symbol of the option.
}

// IsValid checks if the OptionLookupResponse is valid.
//
// Returns:
//   - bool: True if the OptionLookupResponse has a non-empty OptionSymbol, otherwise false.
func (olr *OptionLookupResponse) IsValid() bool {
	return olr.OptionSymbol != ""
}

// String returns a string representation of the OptionLookupResponse.
//
// Returns:
//   - string: A string that represents the OptionLookupResponse, including the OptionSymbol.
func (olr *OptionLookupResponse) String() string {
	return fmt.Sprintf("OptionLookupResponse{OptionSymbol: %q}", olr.OptionSymbol)
}

// Unpack validates the OptionLookupResponse and returns the OptionSymbol if valid.
//
// Returns:
//   - string: The OptionSymbol of the OptionLookupResponse if it is valid.
//   - error: An error if the OptionLookupResponse is invalid.
func (olr *OptionLookupResponse) Unpack() (string, error) {
	if !olr.IsValid() {
		return "", fmt.Errorf("invalid OptionLookupResponse")
	}
	return olr.OptionSymbol, nil
}
