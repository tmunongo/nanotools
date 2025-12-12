package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/services"
	"github.com/tmunongo/nanotools/web/templates/components"
	"github.com/tmunongo/nanotools/web/templates/tools"
)

func Base64PageHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.Base64Page().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// Base64EncodeHandler handles Base64 encoding requests
func Base64EncodeHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		if err := r.ParseForm(); err != nil {
			renderBase64Error(w, r.Context(), "Invalid form data")
			return
		}

		input := r.FormValue("input")
		if input == "" {
			renderBase64Error(w, r.Context(), "Input is required")
			return
		}

		// Encode the input
		output := services.EncodeBase64(input)

		// Log to audit
		processingTime := time.Since(startTime).Milliseconds()
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "base64_encode",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			InputSizeBytes:   sql.NullInt64{Int64: int64(len(input)), Valid: true},
			OutputSizeBytes:  sql.NullInt64{Int64: int64(len(output)), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
			Status:           "success",
		})

		// Render output
		err := components.Base64Output(output, "Encoding").Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render output", http.StatusInternalServerError)
		}
	}
}

// Base64DecodeHandler handles Base64 decoding requests
func Base64DecodeHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		if err := r.ParseForm(); err != nil {
			renderBase64Error(w, r.Context(), "Invalid form data")
			return
		}

		input := r.FormValue("input")
		if input == "" {
			renderBase64Error(w, r.Context(), "Input is required")
			return
		}

		// Decode the input
		output, err := services.DecodeBase64(input)

		// Determine status for logging
		status := "success"
		var errorMsg sql.NullString

		if err != nil {
			status = "error"
			errorMsg = sql.NullString{String: err.Error(), Valid: true}

			// Log the error attempt
			processingTime := time.Since(startTime).Milliseconds()
			_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
				ToolName:         "base64_decode",
				IpAddress:        r.RemoteAddr,
				UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
				InputSizeBytes:   sql.NullInt64{Int64: int64(len(input)), Valid: true},
				ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
				Status:           status,
				ErrorMessage:     errorMsg,
			})

			renderBase64Error(w, r.Context(), err.Error())
			return
		}

		// Log successful decode
		processingTime := time.Since(startTime).Milliseconds()
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "base64_decode",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			InputSizeBytes:   sql.NullInt64{Int64: int64(len(input)), Valid: true},
			OutputSizeBytes:  sql.NullInt64{Int64: int64(len(output)), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
			Status:           status,
		})

		// Render output
		err = components.Base64Output(output, "Decoding").Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render output", http.StatusInternalServerError)
		}
	}
}

func renderBase64Error(w http.ResponseWriter, ctx context.Context, errMsg string) {
	_ = components.JSONOutput("", false, errMsg).Render(ctx, w)
}
