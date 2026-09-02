package params

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

var (
	d1 = time.Date(2024, 1, 2, 9, 30, 0, 0, time.UTC) // time-of-day should be ignored
	d2 = time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
)

func TestWindow_Apply(t *testing.T) {
	cases := []struct {
		name string
		w    Window
		want url.Values
	}{
		{"zero", Window{}, url.Values{}},
		{"OnDate", OnDate(d1), url.Values{"date": {"2024-01-02"}}},
		{"Between", Between(d1, d2), url.Values{"from": {"2024-01-02"}, "to": {"2024-01-31"}}},
		{"Since", Since(d1), url.Values{"from": {"2024-01-02"}}},
		{"Until", Until(d2), url.Values{"to": {"2024-01-31"}}},
		{"LastN", LastN(30), url.Values{"countback": {"30"}}},
		{"LastNUntil", LastNUntil(30, d2), url.Values{"countback": {"30"}, "to": {"2024-01-31"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := url.Values{}
			tc.w.Apply(got)
			if got.Encode() != tc.want.Encode() {
				t.Errorf("Apply() = %q, want %q", got.Encode(), tc.want.Encode())
			}
		})
	}
}

func TestWindow_Validate_OK(t *testing.T) {
	valid := []Window{
		{}, // zero window is valid
		OnDate(d1),
		Between(d1, d2),
		Between(d1, d1), // equal bounds allowed
		Since(d1),
		Until(d2),
		LastN(1),
		LastNUntil(5, d2),
	}
	for i, w := range valid {
		if err := w.Validate(); err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
		}
	}
}

func TestWindow_Validate_Errors(t *testing.T) {
	cases := []struct {
		name string
		w    Window
	}{
		{"date zero", OnDate(time.Time{})},
		{"from zero in range", Between(time.Time{}, d2)},
		{"to zero in range", Between(d1, time.Time{})},
		{"from after to", Between(d2, d1)},
		{"since zero", Since(time.Time{})},
		{"until zero", Until(time.Time{})},
		{"countback zero", LastN(0)},
		{"countback negative", LastN(-3)},
		{"countbackto zero countback", LastNUntil(0, d2)},
		{"countbackto zero to", LastNUntil(5, time.Time{})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.w.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var ve *sdkerrors.ValidationError
			if !errors.As(err, &ve) {
				t.Errorf("error is %T, want *sdkerrors.ValidationError", err)
			}
		})
	}
}

func TestWindow_Accessors(t *testing.T) {
	w := Between(d1, d2)
	if w.Kind() != KindFromTo {
		t.Errorf("Kind() = %v, want KindFromTo", w.Kind())
	}
	if w.IsZero() {
		t.Error("IsZero() = true, want false")
	}
	if !w.IsRange() {
		t.Error("IsRange() = false, want true")
	}
	if !w.From().Equal(d1) || !w.To().Equal(d2) {
		t.Errorf("From/To = %v/%v", w.From(), w.To())
	}
	if (Window{}).IsZero() != true {
		t.Error("zero window IsZero() = false")
	}
	if OnDate(d1).IsRange() {
		t.Error("OnDate IsRange() = true, want false")
	}
	if got := LastN(7).Countback(); got != 7 {
		t.Errorf("Countback() = %d, want 7", got)
	}
	if got := OnDate(d1).Date(); !got.Equal(d1) {
		t.Errorf("Date() = %v, want %v", got, d1)
	}
}

func TestWindow_Chunk(t *testing.T) {
	full := Between(d1, d2)
	mid := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	c := full.Chunk(d1, mid)
	if c.Kind() != KindFromTo || !c.From().Equal(d1) || !c.To().Equal(mid) {
		t.Errorf("Chunk() = %+v", c)
	}
}

func TestWindow_AnchorCountback(t *testing.T) {
	anchor := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)

	// A bare countback becomes the equivalent countback-ending-at-anchor.
	got := LastN(4).AnchorCountback(anchor)
	if got.Kind() != KindCountbackTo || got.Countback() != 4 || !got.To().Equal(anchor) {
		t.Errorf("AnchorCountback(LastN(4)) = %+v, want countback=4 to=%v", got, anchor)
	}

	// Every other kind passes through unchanged.
	others := []Window{
		{},
		OnDate(d1),
		Between(d1, d2),
		Since(d1),
		Until(d2),
		LastNUntil(3, d2),
	}
	for _, w := range others {
		if got := w.AnchorCountback(anchor); got != w {
			t.Errorf("AnchorCountback(%+v) = %+v, want unchanged", w, got)
		}
	}
}

func TestIntToStr(t *testing.T) {
	cases := map[int]string{0: "0", 5: "5", 42: "42", 100: "100", -7: "-7"}
	for in, want := range cases {
		if got := intToStr(in); got != want {
			t.Errorf("intToStr(%d) = %q, want %q", in, got, want)
		}
	}
}
