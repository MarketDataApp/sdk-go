package marketdata

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// canonicalHandler is a [slog.Handler] that emits the canonical Market Data
// log format shared by every official SDK (SDK requirements §7, ADR-009):
//
//	{timestamp} - {logger_name} - {level} - {message} [key=value ...]
//
// For example:
//
//	2026-07-29 10:15:04 - marketdata.client - INFO - client initialized
//
// It backs the SDK's default logger only: a logger injected with
// [WithLogger] keeps whatever format its own handler produces.
type canonicalHandler struct {
	mu     *sync.Mutex // shared across WithAttrs/WithGroup copies
	w      io.Writer
	level  slog.Leveler
	name   string
	prefix string // group prefix applied to attribute keys
	attrs  string // preformatted " key=value" pairs from WithAttrs
}

// newCanonicalHandler creates a handler writing to w, filtering below level.
func newCanonicalHandler(w io.Writer, level slog.Leveler) *canonicalHandler {
	return &canonicalHandler{
		mu:    &sync.Mutex{},
		w:     w,
		level: level,
		name:  "marketdata.client",
	}
}

// Enabled implements [slog.Handler].
func (h *canonicalHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// canonicalLevelName maps slog levels to the level names the canonical
// format uses across SDKs; slog spells WARNING as "WARN".
func canonicalLevelName(l slog.Level) string {
	if l == slog.LevelWarn {
		return "WARNING"
	}
	return l.String()
}

// Handle implements [slog.Handler].
func (h *canonicalHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Time.Format("2006-01-02 15:04:05"))
	b.WriteString(" - ")
	b.WriteString(h.name)
	b.WriteString(" - ")
	b.WriteString(canonicalLevelName(r.Level))
	b.WriteString(" - ")
	b.WriteString(r.Message)
	b.WriteString(h.attrs)
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(&b, " %s%s=%v", h.prefix, a.Key, a.Value)
		return true
	})
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write([]byte(b.String()))
	return err
}

// WithAttrs implements [slog.Handler].
func (h *canonicalHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	var b strings.Builder
	b.WriteString(h.attrs)
	for _, a := range attrs {
		fmt.Fprintf(&b, " %s%s=%v", h.prefix, a.Key, a.Value)
	}
	h2.attrs = b.String()
	return &h2
}

// WithGroup implements [slog.Handler].
func (h *canonicalHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	h2.prefix = h.prefix + name + "."
	return &h2
}
