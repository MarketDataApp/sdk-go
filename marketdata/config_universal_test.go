package marketdata

import "testing"

func TestUniversalParamOptions(t *testing.T) {
	cfg := &Config{}

	WithColumns("last", "bid", "ask").apply(cfg)
	if got := cfg.defaultParams.Get("columns"); got != "last,bid,ask" {
		t.Errorf("columns = %q, want last,bid,ask", got)
	}

	WithDateFormat("unix").apply(cfg)
	if got := cfg.defaultParams.Get("dateformat"); got != "unix" {
		t.Errorf("dateformat = %q, want unix", got)
	}

	// human is format-only: it renames every field in the response, so it is
	// sent by the CSV/HTML facets and never on a typed JSON request.
	WithHumanReadable(true).apply(cfg)
	if got := cfg.formatOnlyParams.Get("human"); got != "true" {
		t.Errorf("format-only human = %q, want true", got)
	}
	if got := cfg.defaultParams.Get("human"); got != "" {
		t.Errorf("human = %q in the request defaults, want it format-only", got)
	}

	WithAddHeaders(false).apply(cfg)
	if got := cfg.defaultParams.Get("headers"); got != "false" {
		t.Errorf("headers = %q, want false", got)
	}
}

func TestUniversalParamOptions_NoOps(t *testing.T) {
	cfg := &Config{}
	WithColumns().apply(cfg)      // empty varargs: no-op
	WithDateFormat("").apply(cfg) // empty format: no-op
	if len(cfg.defaultParams) != 0 {
		t.Errorf("defaultParams = %v, want empty after no-op options", cfg.defaultParams)
	}
}

func TestModeAndPaginationOptions(t *testing.T) {
	cfg := &Config{}
	WithMode(ModeCached).apply(cfg)
	if got := cfg.defaultParams.Get("mode"); got != "cached" {
		t.Errorf("mode = %q, want cached", got)
	}
	WithMaxAge("5min").apply(cfg)
	if got := cfg.defaultParams.Get("maxage"); got != "5min" {
		t.Errorf("maxage = %q, want 5min", got)
	}
	WithLimit(500).apply(cfg)
	if got := cfg.defaultParams.Get("limit"); got != "500" {
		t.Errorf("limit = %q, want 500", got)
	}
	WithOffset(20).apply(cfg)
	if got := cfg.defaultParams.Get("offset"); got != "20" {
		t.Errorf("offset = %q, want 20", got)
	}
}

func TestModeAndPagination_NoOps(t *testing.T) {
	cfg := &Config{}
	WithMode("").apply(cfg)
	WithMaxAge("").apply(cfg)
	WithLimit(0).apply(cfg)
	WithOffset(-1).apply(cfg)
	if len(cfg.defaultParams) != 0 {
		t.Errorf("defaultParams = %v, want empty after no-op options", cfg.defaultParams)
	}
}

// --- Environment-variable configuration (requirements §4) ---

func TestEnvVar_BaseURLAndAPIVersion(t *testing.T) {
	t.Setenv("MARKETDATA_BASE_URL", "https://mock.example.test")
	t.Setenv("MARKETDATA_API_VERSION", "v9")

	cfg := defaultConfig(nil)
	if cfg.baseURL != "https://mock.example.test" {
		t.Errorf("baseURL = %q, want env override", cfg.baseURL)
	}
	if cfg.apiVersion != "v9" {
		t.Errorf("apiVersion = %q, want v9", cfg.apiVersion)
	}
}

func TestEnvVar_ClientOptionBeatsEnv(t *testing.T) {
	t.Setenv("MARKETDATA_BASE_URL", "https://env.example.test")

	client, err := NewClient(
		WithToken("test-token"),
		WithBaseURL("https://option.example.test"),
		WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	if client.config.baseURL != "https://option.example.test" {
		t.Errorf("baseURL = %q, want the client option to beat the env var (cascade §4)", client.config.baseURL)
	}
}

func TestEnvVar_InvalidBaseURLRejected(t *testing.T) {
	t.Setenv("MARKETDATA_BASE_URL", "not-a-url")

	_, err := NewClient(WithToken("test-token"), WithoutStartupValidation())
	if err == nil {
		t.Fatal("NewClient() with malformed MARKETDATA_BASE_URL should fail constructor validation")
	}
}

func TestEnvVar_Mode(t *testing.T) {
	t.Setenv("MARKETDATA_MODE", "cached")

	cfg := defaultConfig(nil)
	if got := cfg.defaultParams.Get("mode"); got != "cached" {
		t.Errorf("defaultParams mode = %q, want cached", got)
	}
}
