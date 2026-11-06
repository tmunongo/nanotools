package telemetry

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

func TestLoadConfigFromEnv(t *testing.T) {
	// Isolate defaults by resetting all relevant environment variables
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("SERVICE_NAME", "")
	t.Setenv("POSTHOG_LOGS_URL", "")
	t.Setenv("POSTHOG_URL", "")
	t.Setenv("POSTHOG_HOST", "")
	t.Setenv("POSTHOG_API_KEY", "")
	t.Setenv("POSTHOG_TOKEN", "")

	// Test isolated defaults
	cfg := LoadConfigFromEnv()
	if cfg.ServiceName != DefaultServiceName {
		t.Errorf("expected default service name %s, got %s", DefaultServiceName, cfg.ServiceName)
	}
	if cfg.EndpointURL != DefaultPostHogLogsURL {
		t.Errorf("expected default endpoint %s, got %s", DefaultPostHogLogsURL, cfg.EndpointURL)
	}
	if cfg.APIKey != "" {
		t.Errorf("expected empty API key when unset, got %s", cfg.APIKey)
	}

	// Test URL normalization
	t.Setenv("POSTHOG_URL", "https://eu.i.posthog.com")
	cfg = LoadConfigFromEnv()
	if cfg.EndpointURL != "https://eu.i.posthog.com/i/v1/logs" {
		t.Errorf("expected normalized url https://eu.i.posthog.com/i/v1/logs, got %s", cfg.EndpointURL)
	}

	// Test explicit logs url
	t.Setenv("POSTHOG_LOGS_URL", "https://eu.i.posthog.com/i/v1/logs")
	cfg = LoadConfigFromEnv()
	if cfg.EndpointURL != "https://eu.i.posthog.com/i/v1/logs" {
		t.Errorf("expected logs url, got %s", cfg.EndpointURL)
	}

	// Test custom API key via POSTHOG_API_KEY
	t.Setenv("POSTHOG_API_KEY", "custom-ph-key")
	cfg = LoadConfigFromEnv()
	if cfg.APIKey != "custom-ph-key" {
		t.Errorf("expected custom API key, got %s", cfg.APIKey)
	}

	// Test custom API key via POSTHOG_TOKEN
	t.Setenv("POSTHOG_API_KEY", "")
	t.Setenv("POSTHOG_TOKEN", "token-from-env")
	cfg = LoadConfigFromEnv()
	if cfg.APIKey != "token-from-env" {
		t.Errorf("expected custom API key from POSTHOG_TOKEN, got %s", cfg.APIKey)
	}
}

func TestSetupLoggingProvider(t *testing.T) {
	// Restore global logger provider on cleanup
	prevProvider := otel.GetLoggerProvider()
	t.Cleanup(func() {
		otel.SetLoggerProvider(prevProvider)
	})

	ctx := context.Background()
	provider, err := SetupLogging(ctx, Config{
		ServiceName: "test-app",
		EndpointURL: "https://us.i.posthog.com/i/v1/logs",
		APIKey:      "phc_test123",
	})
	if err != nil {
		t.Fatalf("SetupLogging failed: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider to be non-nil")
	}

	// Ensure provider shutdown runs even if subsequent assertions fail
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = provider.Shutdown(shutdownCtx)
	})

	logger := Logger()
	if logger == nil {
		t.Fatal("expected Logger() to return non-nil logger")
	}
}

func TestSetupLoggingNoOpWhenTokenMissing(t *testing.T) {
	prevProvider := otel.GetLoggerProvider()
	t.Cleanup(func() {
		otel.SetLoggerProvider(prevProvider)
	})

	t.Setenv("POSTHOG_API_KEY", "")
	t.Setenv("POSTHOG_TOKEN", "")

	ctx := context.Background()
	provider, err := SetupLogging(ctx, Config{
		ServiceName: "test-noop",
		APIKey:      "",
	})
	if err != nil {
		t.Fatalf("expected no error when token is missing, got: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil no-op provider")
	}

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = provider.Shutdown(shutdownCtx)
	})

	logger := Logger()
	if logger == nil {
		t.Fatal("expected Logger() to return non-nil logger")
	}
}
