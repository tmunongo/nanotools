package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/handlers"
	custommw "github.com/tmunongo/nanotools/internal/middleware"
)

func main() {
	// Initialize database
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/tinyutils.db"
	}

	database, queries, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	log.Printf("Database initialized at %s", dbPath)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(custommw.SecureHeaders)

	r.Use(custommw.MaxBytesMiddleware(50 * 1024 * 1024))

	rateLimiter := custommw.NewRateLimiter(10.0, 20)
	r.Use(rateLimiter.Middleware)

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

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Starting nanotools on %s\n", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		fmt.Fprintf(os.Stderr, "Server failed to start: %v\n", err)
		os.Exit(1)
	}
}
