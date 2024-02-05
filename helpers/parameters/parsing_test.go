package parameters

import (
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

func TestParseAndSetParams(t *testing.T) {
	// Create a new instance of SymbolParams with a valid symbol
	OptionParams := &OptionParams{}
	OptionParams.SetExpiration("2020-01-01")

	// Create a new resty request for testing
	client := resty.New()
	request := client.R()

	// Call ParseAndSetParams with the SymbolParams instance and the test request
	err := ParseAndSetParams(OptionParams, request)
	assert.NoError(t, err, "ParseAndSetParams should not return an error")

	// Verify that the symbol parameter was set correctly
	assert.Equal(t, "2020-01-01", request.QueryParam.Get("expiration"), "The symbol parameter was not set correctly")
}
