// Pure rendering helpers for the detail pane: sparkline turns a slice of
// closes into a compact Unicode block chart, and rangeBar renders the
// 52-week low/high/last indicator. Neither function performs I/O or
// depends on styles.go (task 2.5 owns color); both are exercised directly
// by unit tests and consumed by the views built in task 2.5.
package main

import "strings"

// sparkBlocks are the eight Unicode block levels sparkline scales values
// across, from lowest (▁) to highest (█).
var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// sparkMidBlock is the level sparkline uses for a degenerate (flat or
// single-point) series, where there is no range to scale against.
const sparkMidBlock = 3 // ▄

// sparkline renders closes as a compact Unicode block-character chart no
// wider than width. The minimum value in closes maps to the lowest block
// (▁) and the maximum to the highest (█), with every other value scaled
// linearly between them; a flat series (every value equal, including a
// single-point series) renders every point as the middle block (▄). An
// empty slice or a non-positive width returns "".
//
// When closes has more points than width, sparkline first downsamples by
// averaging closes into width contiguous buckets, so the rendered string
// is exactly width runes. When closes has fewer points than width,
// sparkline renders exactly one rune per point, so the result can be
// shorter than width — it never pads.
func sparkline(closes []float64, width int) string {
	if len(closes) == 0 || width <= 0 {
		return ""
	}

	values := closes
	if len(closes) > width {
		values = downsample(closes, width)
	}

	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	var b strings.Builder
	b.Grow(len(values) * 3) // block chars are 3 bytes each in UTF-8
	for _, v := range values {
		b.WriteRune(sparkBlockFor(v, min, max))
	}
	return b.String()
}

// sparkBlockFor maps v, scaled linearly against [min, max], to one of the
// eight sparkBlocks levels. A degenerate range (min == max) always
// returns sparkMidBlock, since there is nothing to scale against.
func sparkBlockFor(v, min, max float64) rune {
	if min == max {
		return sparkBlocks[sparkMidBlock]
	}
	frac := (v - min) / (max - min)
	idx := int(frac * float64(len(sparkBlocks)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sparkBlocks) {
		idx = len(sparkBlocks) - 1
	}
	return sparkBlocks[idx]
}

// downsample reduces values to exactly width points by averaging each of
// width contiguous, roughly equal-sized buckets. Callers must only invoke
// it when len(values) > width, so every bucket has at least one source
// point.
func downsample(values []float64, width int) []float64 {
	n := len(values)
	out := make([]float64, width)
	for i := 0; i < width; i++ {
		lo := i * n / width
		hi := (i + 1) * n / width
		if hi <= lo {
			hi = lo + 1
		}
		sum := 0.0
		for _, v := range values[lo:hi] {
			sum += v
		}
		out[i] = sum / float64(hi-lo)
	}
	return out
}

// rangeBar renders a 52-week range indicator of exactly width runes: a
// run of '─' with a single '●' marker positioned proportionally between
// low and high at last's value, e.g. "────●────". last is clamped into
// [low, high] before positioning, so a current price outside the 52-week
// range still renders at an edge rather than off the bar. A degenerate
// range (low == high) centers the marker. Returns "" when width < 3 or
// the range is invalid (high < low).
func rangeBar(low, high, last float64, width int) string {
	if width < 3 || high < low {
		return ""
	}

	if last < low {
		last = low
	}
	if last > high {
		last = high
	}

	pos := (width - 1) / 2 // degenerate low == high: centered
	if high > low {
		frac := (last - low) / (high - low)
		pos = int(frac * float64(width-1))
		if pos < 0 {
			pos = 0
		}
		if pos >= width {
			pos = width - 1
		}
	}

	runes := make([]rune, width)
	for i := range runes {
		runes[i] = '─'
	}
	runes[pos] = '●'
	return string(runes)
}
