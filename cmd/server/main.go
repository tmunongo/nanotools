package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/handlers"
	custommw "github.com/tmunongo/nanotools/internal/middleware"
	"github.com/tmunongo/nanotools/internal/telemetry"
)

func run() error {
	ctx := context.Background()

	// Initialize PostHog OpenTelemetry telemetry
	provider, err := telemetry.SetupLogging(ctx)
	if err != nil {
		log.Printf("Warning: failed to initialize telemetry: %v", err)
		provider = sdklog.NewLoggerProvider()
		otel.SetLoggerProvider(provider)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(shutdownCtx); err != nil {
			log.Printf("Failed to shut down telemetry provider: %v", err)
		}
	}()

	// Initialize database
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/tinyutils.db"
	}

	database, queries, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	log.Printf("Database initialized at %s", dbPath)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Block automated vulnerability & credentials scanners (.env, .git, .php, etc.)
	// Placed before logging and template rendering to prevent log spam and resource waste.
	r.Use(custommw.BlockBotScanners)

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(custommw.SecureHeaders)

	r.Use(custommw.MaxBytesMiddleware(50 * 1024 * 1024))

	rateLimiter := custommw.NewRateLimiter(10.0, 20)
	r.Use(rateLimiter.Middleware)

	videoDownloadLimiter := custommw.NewRateLimiter(1.0, 3)

	// routes
	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	r.Get("/", handlers.HomeHandler)

	r.Get("/tools/json-formatter", handlers.JSONFormatterHandler)

	r.Post("/api/tools/json-format", handlers.JSONFormatAPIHandler(queries))

	r.Get("/tools/base64", handlers.Base64PageHandler)
	r.Post("/api/tools/base64/encode", handlers.Base64EncodeHandler(queries))
	r.Post("/api/tools/base64/decode", handlers.Base64DecodeHandler(queries))

	r.Get("/tools/uuid", handlers.UUIDPageHandler)
	r.Get("/api/tools/uuid/generate", handlers.UUIDGenerateHandler(queries))

	r.Get("/tools/qr-code", handlers.QRCodePageHandler)
	r.Post("/api/tools/qr/generate", handlers.QRCodeGenerateHandler(queries))

	r.Get("/tools/slugify", handlers.SlugifyPageHandler)
	r.Post("/api/tools/slugify", handlers.SlugifyAPIHandler(queries))

	r.Get("/tools/image-converter", handlers.ImageConverterPageHandler)
	r.Post("/api/tools/image/convert", handlers.ImageConvertHandler(queries))

	r.Get("/tools/video-downloader", handlers.VideoDownloaderPageHandler)
	r.Post("/api/tools/video/info", handlers.VideoInfoHandler(queries))
	r.With(videoDownloadLimiter.Middleware).Post("/api/tools/video/download", handlers.VideoDownloadHandler(queries))

	r.Get("/tools/pdf-to-images", handlers.PDFConverterPageHandler)
	r.Post("/api/tools/pdf/to-images", handlers.PDFToImagesHandler(queries))

	r.Get("/tools/pdf-joiner", handlers.PDFJoinerPageHandler)
	r.Post("/api/tools/pdf/join", handlers.JoinPDFsHandler(queries))

	r.Get("/tools/pdf-unlocker", handlers.PDFUnlockerPageHandler)
	r.Post("/api/tools/pdf/unlock", handlers.UnlockPDFHandler(queries))

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.NotFound(handlers.NotFoundHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	portNum, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		portNum = 8080
	}

	// Emit application startup log to PostHog OpenTelemetry logger using configured service name
	telemetry.EmitLog(ctx, otellog.SeverityInfo, "Application started", attribute.Int64("server.port", portNum))

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Starting nanotools on %s\n", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		return fmt.Errorf("server failed to start: %w", err)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
