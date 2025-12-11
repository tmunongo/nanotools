package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/tmunongo/nanotools/internal/handlers"
	custommw "github.com/tmunongo/nanotools/internal/middleware"
)

func main() {
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

	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	r.Get("/", handlers.HomeHandler)

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