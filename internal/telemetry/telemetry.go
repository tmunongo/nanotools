package telemetry

import (
	"context"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	DefaultServiceName    = "nanotools"
	DefaultPostHogLogsURL = "https://us.i.posthog.com/i/v1/logs"
)

var (
	serviceNameMu      sync.RWMutex
	currentServiceName = DefaultServiceName
)

func getServiceName() string {
	serviceNameMu.RLock()
	defer serviceNameMu.RUnlock()
	return currentServiceName
}

func setServiceName(name string) {
	serviceNameMu.Lock()
	defer serviceNameMu.Unlock()
	currentServiceName = name
}

// Config holds settings for telemetry log export.
type Config struct {
	ServiceName string
	EndpointURL string
	APIKey      string
}

// LoadEnvFile reads a simple .env file and sets any environment variables
// that are not already set in the current process.
func LoadEnvFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

// LoadConfigFromEnv builds a telemetry Config from environment variables
// with sensible defaults for PostHog.
func LoadConfigFromEnv() Config {
	// Attempt to load .env file if present
	LoadEnvFile(".env")

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = os.Getenv("SERVICE_NAME")
	}
	if serviceName == "" {
		serviceName = DefaultServiceName
	}

	endpoint := os.Getenv("POSTHOG_LOGS_URL")
	if endpoint == "" {
		endpoint = os.Getenv("POSTHOG_URL")
	}
	if endpoint == "" {
		endpoint = os.Getenv("POSTHOG_HOST")
	}
	if endpoint == "" {
		endpoint = DefaultPostHogLogsURL
	} else {
		endpoint = strings.TrimRight(endpoint, "/")
		if !strings.HasSuffix(endpoint, "/i/v1/logs") {
			endpoint += "/i/v1/logs"
		}
	}

	apiKey := os.Getenv("POSTHOG_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("POSTHOG_TOKEN")
	}

	return Config{
		ServiceName: serviceName,
		EndpointURL: endpoint,
		APIKey:      apiKey,
	}
}

// SetupLogging initializes OpenTelemetry logging with PostHog OTLP HTTP exporter
// and registers the provider globally. If the PostHog token is absent, it configures
// a no-op provider so telemetry sends no data.
func SetupLogging(ctx context.Context, customCfg ...Config) (*sdklog.LoggerProvider, error) {
	cfg := LoadConfigFromEnv()
	if len(customCfg) > 0 {
		if customCfg[0].ServiceName != "" {
			cfg.ServiceName = customCfg[0].ServiceName
		}
		if customCfg[0].EndpointURL != "" {
			cfg.EndpointURL = customCfg[0].EndpointURL
		}
		if customCfg[0].APIKey != "" {
			cfg.APIKey = customCfg[0].APIKey
		}
	}

	setServiceName(cfg.ServiceName)

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
	)

	// If no token is configured, configure a no-op provider that performs no exports
	if cfg.APIKey == "" {
		provider := sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
		)
		otel.SetLoggerProvider(provider)
		return provider, nil
	}

	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(cfg.EndpointURL),
		otlploghttp.WithHeaders(map[string]string{
			"Authorization": "Bearer " + cfg.APIKey,
		}),
	)
	if err != nil {
		return nil, err
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	otel.SetLoggerProvider(provider)
	return provider, nil
}

// Logger returns the global logger for the current service name.
func Logger() otellog.Logger {
	return otel.Logger(getServiceName())
}

// EmitLog is a helper to quickly emit an OpenTelemetry log record.
func EmitLog(ctx context.Context, severity otellog.Severity, message string, attrs ...attribute.KeyValue) {
	logger := Logger()
	var record otellog.Record
	record.SetSeverity(severity)
	record.SetBody(attribute.StringValue(message))
	if len(attrs) > 0 {
		record.AddAttributes(attrs...)
	}
	logger.Emit(ctx, record)
}
