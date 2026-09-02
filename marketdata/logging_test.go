package marketdata

import (
	"bytes"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

func TestCanonicalHandler_Format(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newCanonicalHandler(&buf, slog.LevelInfo))

	logger.Info("client initialized", "base_url", "https://api.example.test")

	got := buf.String()
	// {timestamp} - {logger_name} - {level} - {message} key=value
	want := regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} - marketdata\.client - INFO - client initialized base_url=https://api\.example\.test\n$`)
	if !want.MatchString(got) {
		t.Errorf("log line = %q, want canonical format %s", got, want)
	}
}

func TestCanonicalHandler_WarningLevelName(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newCanonicalHandler(&buf, slog.LevelInfo))

	logger.Warn("demo mode")

	if !strings.Contains(buf.String(), " - WARNING - ") {
		t.Errorf("log line = %q, want canonical WARNING (not slog's WARN)", buf.String())
	}
}

func TestCanonicalHandler_WithAttrsAndGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newCanonicalHandler(&buf, slog.LevelInfo))

	logger.With("request_id", "abc").WithGroup("retry").Info("retrying", "attempt", 2)

	got := buf.String()
	if !strings.Contains(got, "retrying request_id=abc retry.attempt=2") {
		t.Errorf("log line = %q, want preformatted attr and group-prefixed attr", got)
	}
}

func TestCanonicalHandler_EmptyGroupIsNoop(t *testing.T) {
	h := newCanonicalHandler(&bytes.Buffer{}, slog.LevelInfo)
	if h.WithGroup("") != slog.Handler(h) {
		t.Error("WithGroup(\"\") should return the same handler")
	}
}

func TestCanonicalHandler_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newCanonicalHandler(&buf, slog.LevelInfo))

	logger.Debug("hidden")
	if buf.Len() != 0 {
		t.Errorf("DEBUG below INFO threshold should be suppressed, got %q", buf.String())
	}

	logger.Error("shown")
	if !strings.Contains(buf.String(), " - ERROR - shown") {
		t.Errorf("ERROR should pass the INFO threshold, got %q", buf.String())
	}
}

func TestDefaultLogger_IsCanonicalWithDynamicLevel(t *testing.T) {
	cfg := defaultConfig(nil)

	if cfg.logLevel == nil {
		t.Fatal("default config should carry a dynamic log level")
	}
	if cfg.logLevel.Level() != slog.LevelInfo {
		t.Errorf("default level = %v, want INFO", cfg.logLevel.Level())
	}
	if _, ok := cfg.logger.Handler().(*canonicalHandler); !ok {
		t.Errorf("default handler = %T, want *canonicalHandler", cfg.logger.Handler())
	}
}

func TestWithDebug_RaisesDefaultLoggerLevel(t *testing.T) {
	client, err := NewClient(
		WithToken("test-token"),
		WithoutStartupValidation(),
		WithDebug(true),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	if got := client.config.logLevel.Level(); got != slog.LevelDebug {
		t.Errorf("level with WithDebug(true) = %v, want DEBUG", got)
	}
}

func TestClientDebug_TogglesDefaultLoggerLevel(t *testing.T) {
	client, err := NewClient(WithToken("test-token"), WithoutStartupValidation())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	client.Debug(true)
	if got := client.config.logLevel.Level(); got != slog.LevelDebug {
		t.Errorf("level after Debug(true) = %v, want DEBUG", got)
	}

	client.Debug(false)
	if got := client.config.logLevel.Level(); got != client.config.baseLogLevel {
		t.Errorf("level after Debug(false) = %v, want base %v", got, client.config.baseLogLevel)
	}
}

func TestWithLogger_OwnsItsLevel(t *testing.T) {
	custom := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	client, err := NewClient(
		WithToken("test-token"),
		WithoutStartupValidation(),
		WithLogger(custom),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	if client.config.logLevel != nil {
		t.Error("logLevel should be nil with an injected logger")
	}
	// Must not panic and must leave the injected logger untouched.
	client.Debug(true)
	client.Debug(false)
	if client.config.logger != custom {
		t.Error("injected logger should be preserved")
	}
}
