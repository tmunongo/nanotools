package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/services"
	"github.com/tmunongo/nanotools/web/templates/tools"
)

// SlugifyPageHandler serves the slugify tool page
func SlugifyPageHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.SlugifyPage().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// SlugifyAPIHandler handles slugification requests
func SlugifyAPIHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		input := r.FormValue("input")
		separator := r.FormValue("separator")
		lowercaseStr := r.FormValue("lowercase")
		maxLengthStr := r.FormValue("max_length")

		// Parse options
		lowercase := lowercaseStr == "on"
		maxLength, _ := strconv.Atoi(maxLengthStr)
		if maxLength == 0 {
			maxLength = 80
		}

		// Generate slug
		result := services.Slugify(input, services.SlugifyOptions{
			Separator: separator,
			Lowercase: lowercase,
			MaxLength: maxLength,
		})

		// Log to audit
		processingTime := time.Since(startTime).Milliseconds()
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "slugify",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			InputSizeBytes:   sql.NullInt64{Int64: int64(len(input)), Valid: true},
			OutputSizeBytes:  sql.NullInt64{Int64: int64(len(result)), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
			Status:           "success",
		})

		// Render output
		err := tools.SlugOutput(result).Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render output", http.StatusInternalServerError)
		}
	}
}
