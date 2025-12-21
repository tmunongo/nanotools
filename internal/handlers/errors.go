package handlers

import (
	"net/http"

	"github.com/tmunongo/nanotools/web/templates"
)

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := templates.ErrorPage(
		404,
		"Page Not Found",
		"The page you're looking for doesn't exist. Maybe it's taking a coffee break?",
	).Render(r.Context(), w)

	if err != nil {
		http.Error(w, "404 Not Found", http.StatusNotFound)
	}
}

func InternalErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		println("Internal error:", err.Error())
	}

	w.WriteHeader(http.StatusInternalServerError)
	renderErr := templates.ErrorPage(
		500,
		"Internal Server Error",
		"Something unexpected happened. We're looking into it.",
	).Render(r.Context(), w)

	if renderErr != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
	}
}

// RateLimitErrorHandler serves a rate limit exceeded page
func RateLimitErrorHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTooManyRequests)
	err := templates.RateLimitError().Render(r.Context(), w)

	if err != nil {
		http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
	}
}
