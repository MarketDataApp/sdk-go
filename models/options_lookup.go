package models

import (
	"fmt"
)

type OptionLookupResponse struct {
	OptionSymbol string `json:"optionSymbol"`
}

func (olr *OptionLookupResponse) IsValid() bool {
	if olr.OptionSymbol == "" {
		return false
	}
	return true
}

func (olr *OptionLookupResponse) String() string {
	return fmt.Sprintf("OptionSymbol: %v", olr.OptionSymbol)
}

func (olr *OptionLookupResponse) Unpack() (string, error) {
	if !olr.IsValid() {
		return "", fmt.Errorf("invalid OptionLookupResponse")
	}
	return olr.OptionSymbol, nil
}
