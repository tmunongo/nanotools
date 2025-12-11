package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/services"
	"github.com/tmunongo/nanotools/web/templates/components"
	"github.com/tmunongo/nanotools/web/templates/tools"
)

func JSONFormatterHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.JSONFormatterPage().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func JSONFormatAPIHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		if err := r.ParseForm(); err != nil {
			renderJSONError(w, r.Context(), "Invalid form data")
			return
		}

		input := r.FormValue("input")

		indent, _ := strconv.Atoi(r.FormValue("indent"))
		if indent < 0 || indent > 8 {
			indent = 2
		}

		sortKeys := r.FormValue("sort_keys") == "on"

		result := services.FormatJSON(input, services.JSONFormatterOptions{
			Indent:   indent,
			SortKeys: sortKeys,
		})

		processingTime := time.Since(startTime).Milliseconds()
		status := "success"
		var errorMsg sql.NullString

		if !result.IsValid {
			status = "error"
			errorMsg = sql.NullString{String: result.Error, Valid: true}
		}

		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "json_formatter",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			InputSizeBytes:   sql.NullInt64{Int64: int64(len(input)), Valid: true},
			OutputSizeBytes:  sql.NullInt64{Int64: int64(len(result.Formatted)), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
			Status:           status,
			ErrorMessage:     errorMsg,
		})

		// Render output
		err := components.JSONOutput(result.Formatted, result.IsValid, result.Error).
			Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render output", http.StatusInternalServerError)
		}
	}
}

func renderJSONError(w http.ResponseWriter, ctx context.Context, errMsg string) {
	_ = components.JSONOutput("", false, errMsg).Render(ctx, w)
}
