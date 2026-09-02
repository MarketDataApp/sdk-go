package main

import (
	"testing"
	"time"
)

func TestParseFlagsDefaults(t *testing.T) {
	cfg := parseFlags(nil)

	if cfg.symbol != "AAPL" {
		t.Errorf("symbol = %q, want %q", cfg.symbol, "AAPL")
	}
	if cfg.refresh != 15*time.Second {
		t.Errorf("refresh = %v, want %v", cfg.refresh, 15*time.Second)
	}
	if cfg.once {
		t.Errorf("once = true, want false")
	}
	if cfg.baseURL != "" {
		t.Errorf("baseURL = %q, want empty", cfg.baseURL)
	}
}

func TestParseFlagsSymbolIsUppercased(t *testing.T) {
	cfg := parseFlags([]string{"-symbol", "msft"})

	if cfg.symbol != "MSFT" {
		t.Errorf("symbol = %q, want %q", cfg.symbol, "MSFT")
	}
}

func TestParseFlagsRefreshParses(t *testing.T) {
	cfg := parseFlags([]string{"-refresh", "30s"})

	if cfg.refresh != 30*time.Second {
		t.Errorf("refresh = %v, want %v", cfg.refresh, 30*time.Second)
	}
}

func TestParseFlagsOnce(t *testing.T) {
	cfg := parseFlags([]string{"-once"})

	if !cfg.once {
		t.Errorf("once = false, want true")
	}
}

func TestParseFlagsBaseURL(t *testing.T) {
	cfg := parseFlags([]string{"-base-url", "http://127.0.0.1:9999"})

	if cfg.baseURL != "http://127.0.0.1:9999" {
		t.Errorf("baseURL = %q, want %q", cfg.baseURL, "http://127.0.0.1:9999")
	}
}

func TestNewClientWithBaseURLSkipsStartupValidation(t *testing.T) {
	// A base URL that nothing listens on. If newClient performed startup
	// validation here, this call would fail; WithoutStartupValidation
	// must be applied so client construction succeeds regardless.
	cfg := appConfig{symbol: "AAPL", refresh: 15 * time.Second, baseURL: "http://127.0.0.1:0"}

	client, err := newClient(cfg)
	if err != nil {
		t.Fatalf("newClient() error = %v, want nil", err)
	}
	defer client.Close()
}

func TestNewClientDefault(t *testing.T) {
	cfg := appConfig{symbol: "AAPL", refresh: 15 * time.Second}

	client, err := newClient(cfg)
	if err != nil {
		t.Fatalf("newClient() error = %v, want nil", err)
	}
	defer client.Close()
}
