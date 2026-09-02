package main

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

func TestComma(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{41203110, "41,203,110"},
		{0, "0"},
		{7, "7"},
		{999, "999"},
		{1000, "1,000"},
		{-1234, "-1,234"},
		{-1000000, "-1,000,000"},
		{123456789, "123,456,789"},
	}
	for _, tc := range cases {
		if got := comma(tc.n); got != tc.want {
			t.Errorf("comma(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"under a minute", 42 * time.Second, "42s"},
		{"minutes and seconds", 252 * time.Second, "4m12s"},
		{"hours minutes seconds", time.Hour + 4*time.Minute + 12*time.Second, "1h4m12s"},
		{"exact minute boundary keeps zero seconds", 60 * time.Second, "1m0s"},
		{"exact hour boundary keeps zero minutes and seconds", time.Hour, "1h0m0s"},
		{"rounds up to nearest second", 41*time.Second + 600*time.Millisecond, "42s"},
		{"rounds down to nearest second", 41*time.Second + 400*time.Millisecond, "41s"},
		{"zero", 0, "0s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatDuration(tc.d); got != tc.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tc.d, got, tc.want)
			}
		})
	}
}

func TestClassify_RateLimitError(t *testing.T) {
	now := fixedNow
	rle := &marketdata.RateLimitError{
		Limit:     100000,
		Remaining: 0,
		ResetAt:   now.Add(4*time.Minute + 12*time.Second),
	}
	got := classify(rle, now)
	want := "rate limited — resets in 4m12s"
	if got != want {
		t.Errorf("classify(RateLimitError) = %q, want %q", got, want)
	}
}

func TestClassify_RateLimitError_Wrapped(t *testing.T) {
	now := fixedNow
	rle := &marketdata.RateLimitError{
		ResetAt: now.Add(90 * time.Second),
	}
	wrapped := fmt.Errorf("fetching quotes: %w", rle)
	got := classify(wrapped, now)
	want := "rate limited — resets in 1m30s"
	if got != want {
		t.Errorf("classify(wrapped RateLimitError) = %q, want %q (errors.As must unwrap)", got, want)
	}
}

func TestClassify_RateLimitError_ResetAtInPastFloorsAtZero(t *testing.T) {
	now := fixedNow
	rle := &marketdata.RateLimitError{
		ResetAt: now.Add(-time.Minute), // already past reset
	}
	got := classify(rle, now)
	want := "rate limited — resets in 0s"
	if got != want {
		t.Errorf("classify(RateLimitError past ResetAt) = %q, want %q", got, want)
	}
}

func TestClassify_AuthenticationError(t *testing.T) {
	ae := &marketdata.AuthenticationError{}
	got := classify(ae, time.Now())
	want := "auth failed — check MARKETDATA_TOKEN"
	if got != want {
		t.Errorf("classify(AuthenticationError) = %q, want %q", got, want)
	}
}

func TestClassify_NetworkError_Timeout(t *testing.T) {
	ne := &marketdata.NetworkError{Timeout: true}
	got := classify(ne, time.Now())
	want := "network timeout — retrying on next tick"
	if got != want {
		t.Errorf("classify(NetworkError timeout) = %q, want %q", got, want)
	}
}

func TestClassify_NetworkError_NonTimeout_FallsThroughToErrorString(t *testing.T) {
	ne := &marketdata.NetworkError{Timeout: false, Temporary: true}
	got := classify(ne, time.Now())
	want := ne.Error()
	if got != want {
		t.Errorf("classify(NetworkError non-timeout) = %q, want err.Error() %q", got, want)
	}
}

func TestClassify_GenericError_FallsThroughToErrorString(t *testing.T) {
	err := errors.New("boom")
	got := classify(err, time.Now())
	if got != "boom" {
		t.Errorf("classify(generic error) = %q, want %q", got, "boom")
	}
}
