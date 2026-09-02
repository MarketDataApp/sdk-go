package main

import (
	"testing"
)

// blockLevel returns the index (0-7) of r within sparkBlocks, or -1 if r
// is not one of the eight block characters. Tests use it to reason about
// relative levels without hard-coding which rune means what.
func blockLevel(r rune) int {
	for i, b := range sparkBlocks {
		if b == r {
			return i
		}
	}
	return -1
}

func TestSparkline_Empty(t *testing.T) {
	if got := sparkline(nil, 10); got != "" {
		t.Errorf("sparkline(nil, 10) = %q, want \"\"", got)
	}
	if got := sparkline([]float64{}, 10); got != "" {
		t.Errorf("sparkline([]float64{}, 10) = %q, want \"\"", got)
	}
}

func TestSparkline_NonPositiveWidth(t *testing.T) {
	closes := []float64{1, 2, 3}
	if got := sparkline(closes, 0); got != "" {
		t.Errorf("sparkline(closes, 0) = %q, want \"\"", got)
	}
	if got := sparkline(closes, -5); got != "" {
		t.Errorf("sparkline(closes, -5) = %q, want \"\"", got)
	}
}

func TestSparkline_MinMaxScaling(t *testing.T) {
	// Min is first, max is last; every other point falls strictly between.
	closes := []float64{10, 20, 30, 40, 50}
	got := []rune(sparkline(closes, 5))
	if len(got) != 5 {
		t.Fatalf("len(sparkline) = %d, want 5", len(got))
	}
	if blockLevel(got[0]) != 0 {
		t.Errorf("first rune (min) = %q, want %q (level 0)", got[0], sparkBlocks[0])
	}
	if blockLevel(got[len(got)-1]) != len(sparkBlocks)-1 {
		t.Errorf("last rune (max) = %q, want %q (level %d)", got[len(got)-1], sparkBlocks[len(sparkBlocks)-1], len(sparkBlocks)-1)
	}
}

func TestSparkline_MonotonicRamp(t *testing.T) {
	closes := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := []rune(sparkline(closes, len(closes)))
	if len(got) != len(closes) {
		t.Fatalf("len(sparkline) = %d, want %d", len(got), len(closes))
	}
	prev := blockLevel(got[0])
	for i := 1; i < len(got); i++ {
		lvl := blockLevel(got[i])
		if lvl < prev {
			t.Errorf("block level decreased at index %d: %d -> %d (non-decreasing ramp required)", i, prev, lvl)
		}
		prev = lvl
	}
}

func TestSparkline_Flat(t *testing.T) {
	closes := []float64{100, 100, 100, 100}
	got := []rune(sparkline(closes, 4))
	if len(got) != 4 {
		t.Fatalf("len(sparkline) = %d, want 4", len(got))
	}
	for i, r := range got {
		if r != sparkBlocks[3] {
			t.Errorf("rune[%d] = %q, want mid block %q", i, r, sparkBlocks[3])
		}
	}
}

func TestSparkline_Single(t *testing.T) {
	got := []rune(sparkline([]float64{42}, 10))
	if len(got) != 1 {
		t.Fatalf("len(sparkline) = %d, want 1", len(got))
	}
	if got[0] != sparkBlocks[3] {
		t.Errorf("single-point rune = %q, want mid block %q", got[0], sparkBlocks[3])
	}
}

func TestSparkline_DownsampleWhenLonger(t *testing.T) {
	closes := make([]float64, 100)
	for i := range closes {
		closes[i] = float64(i)
	}
	got := []rune(sparkline(closes, 10))
	if len(got) != 10 {
		t.Fatalf("len(sparkline) = %d, want 10 (downsampled)", len(got))
	}
	// Downsampled ramp should still be non-decreasing since the source is
	// monotonic.
	prev := blockLevel(got[0])
	for i := 1; i < len(got); i++ {
		lvl := blockLevel(got[i])
		if lvl < prev {
			t.Errorf("downsampled block level decreased at index %d: %d -> %d", i, prev, lvl)
		}
		prev = lvl
	}
}

func TestSparkline_FewerPointsThanWidth(t *testing.T) {
	closes := []float64{5, 10, 15}
	got := sparkline(closes, 20)
	if n := len([]rune(got)); n != 3 {
		t.Fatalf("len(sparkline) = %d, want 3 (one rune per point, shorter than width)", n)
	}
}

func TestSparkline_ExactWidthMatch(t *testing.T) {
	closes := []float64{5, 10, 15}
	got := sparkline(closes, 3)
	if n := len([]rune(got)); n != 3 {
		t.Fatalf("len(sparkline) = %d, want 3", n)
	}
}

func TestRangeBar_CenteredExample(t *testing.T) {
	got := rangeBar(0, 100, 50, 9)
	want := "────●────"
	if got != want {
		t.Errorf("rangeBar(0, 100, 50, 9) = %q, want %q", got, want)
	}
}

func TestRangeBar_ExactWidth(t *testing.T) {
	got := rangeBar(10, 20, 15, 21)
	if n := len([]rune(got)); n != 21 {
		t.Fatalf("len(rangeBar) = %d, want 21", n)
	}
}

func TestRangeBar_ClampBelowLow(t *testing.T) {
	got := []rune(rangeBar(100, 200, 50, 11))
	if got[0] != '●' {
		t.Errorf("rangeBar with last < low: marker at index %d, want 0 (leftmost); bar=%q", indexOfMarker(got), string(got))
	}
}

func TestRangeBar_ClampAboveHigh(t *testing.T) {
	got := []rune(rangeBar(100, 200, 500, 11))
	if got[len(got)-1] != '●' {
		t.Errorf("rangeBar with last > high: marker at index %d, want %d (rightmost); bar=%q", indexOfMarker(got), len(got)-1, string(got))
	}
}

func TestRangeBar_Degenerate(t *testing.T) {
	got := []rune(rangeBar(150, 150, 150, 9))
	if len(got) != 9 {
		t.Fatalf("len(rangeBar) = %d, want 9", len(got))
	}
	wantPos := (9 - 1) / 2
	if got[wantPos] != '●' {
		t.Errorf("degenerate rangeBar marker at index %d, want %d; bar=%q", indexOfMarker(got), wantPos, string(got))
	}
}

func TestRangeBar_InvalidWidth(t *testing.T) {
	if got := rangeBar(0, 100, 50, 2); got != "" {
		t.Errorf("rangeBar with width 2 = %q, want \"\"", got)
	}
	if got := rangeBar(0, 100, 50, 0); got != "" {
		t.Errorf("rangeBar with width 0 = %q, want \"\"", got)
	}
}

func TestRangeBar_InvalidRange(t *testing.T) {
	if got := rangeBar(100, 50, 75, 11); got != "" {
		t.Errorf("rangeBar with high < low = %q, want \"\"", got)
	}
}

// indexOfMarker returns the index of the '●' rune in bar, or -1 if absent.
// Test helper for readable failure messages.
func indexOfMarker(bar []rune) int {
	for i, r := range bar {
		if r == '●' {
			return i
		}
	}
	return -1
}
