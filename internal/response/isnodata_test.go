package response

import "testing"

func TestIsNoData(t *testing.T) {
	cases := map[int]bool{
		404: true,  // no data for request
		204: true,  // mode=cached cache miss
		200: false, // fresh
		203: false, // served from cache, has data
		500: false, // server error
	}
	for code, want := range cases {
		if got := IsNoData(code); got != want {
			t.Errorf("IsNoData(%d) = %v, want %v", code, got, want)
		}
	}
}
