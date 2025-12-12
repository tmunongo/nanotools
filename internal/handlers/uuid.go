package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/services"
	"github.com/tmunongo/nanotools/web/templates/tools"
)

// UUIDPageHandler serves the UUID generator page
func UUIDPageHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.UUIDPage().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// UUIDGenerateHandler handles UUID generation API requests
func UUIDGenerateHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Parse query parameters
		countStr := r.URL.Query().Get("count")
		uppercaseStr := r.URL.Query().Get("uppercase")
		hyphensStr := r.URL.Query().Get("hyphens")

		// Parse count with default
		count, err := strconv.Atoi(countStr)
		if err != nil || count < 1 {
			count = 1
		}

		// Parse boolean flags
		uppercase := uppercaseStr == "true"
		withHyphens := hyphensStr == "true"

		// Generate UUIDs
		uuids := services.GenerateUUIDs(services.UUIDOptions{
			Count:       count,
			Uppercase:   uppercase,
			WithHyphens: withHyphens,
		})

		// Calculate output size (approximate)
		outputSize := 0
		for _, uuid := range uuids {
			outputSize += len(uuid)
		}

		// Log to audit
		processingTime := time.Since(startTime).Milliseconds()
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "uuid_generator",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			InputSizeBytes:   sql.NullInt64{Int64: 0, Valid: true}, // No input
			OutputSizeBytes:  sql.NullInt64{Int64: int64(outputSize), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
			Status:           "success",
		})

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuids": uuids,
			"count": len(uuids),
		})
	}
}
