package client

import (
	"testing"
)

func TestIndexQuoteRequest_Packed(t *testing.T) {
	iqPacked, err := IndexQuotes().Symbol("VIX").FiftyTwoWeek(true).Packed()
	if err != nil {
		t.Errorf("Failed to get index quotes: %v", err)
		return
	}

	iq, err := iqPacked.Unpack()
	if err != nil {
		t.Errorf("Failed to unpack index quotes: %v", err)
		return
	}

	if len(iq) == 0 {
		t.Errorf("Unpacked index quotes slice is empty")
		return
	}

	firstQuote := iq[0]
	if firstQuote.Symbol != "VIX" {
		t.Errorf("Expected symbol VIX, got %s", firstQuote.Symbol)
	}

	if firstQuote.Last < 0 || firstQuote.Last > 100 {
		t.Errorf("Expected last value to be between 0 and 100, got %f", firstQuote.Last)
	}
}

func TestIndexQuoteRequest_Get(t *testing.T) {
	iq, err := IndexQuotes().Symbol("VIX").FiftyTwoWeek(false).Get()
	if err != nil {
		t.Errorf("Failed to get index quotes: %v", err)
		return
	}

	if len(iq) == 0 {
		t.Errorf("Index quotes slice is empty")
		return
	}

	for _, quote := range iq {
		if quote.Symbol != "VIX" {
			t.Errorf("Expected symbol VIX, got %s", quote.Symbol)
		}
		if quote.Last < 0 || quote.Last > 100 {
			t.Errorf("Expected last value to be between 0 and 100, got %f", quote.Last)
		}
	}
}
