package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/services"
	"github.com/tmunongo/nanotools/web/templates/tools"
)

func PDFConverterPageHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.PDFConverterPage().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func PDFToImagesHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		if err := r.ParseMultipartForm(50 << 20); err != nil {
			http.Error(w, "File too large or invalid", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("pdf")
		if err != nil {
			http.Error(w, "No PDF uploaded", http.StatusBadRequest)
			return
		}
		defer file.Close()

		dpiStr := r.FormValue("dpi")
		format := r.FormValue("format")
		qualityStr := r.FormValue("quality")

		dpi, _ := strconv.Atoi(dpiStr)
		quality, _ := strconv.Atoi(qualityStr)

		images, err := services.ConvertPDFToImages(file, services.PDFToImagesOptions{
			DPI:     dpi,
			Format:  format,
			Quality: quality,
		})

		if err != nil {
			_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
				ToolName:         "pdf_to_images",
				IpAddress:        r.RemoteAddr,
				UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
				InputSizeBytes:   sql.NullInt64{Int64: header.Size, Valid: true},
				ProcessingTimeMs: sql.NullInt64{Int64: time.Since(startTime).Milliseconds(), Valid: true},
				Status:           "error",
				ErrorMessage:     sql.NullString{String: err.Error(), Valid: true},
			})

			http.Error(w, fmt.Sprintf("Conversion failed: %v", err), http.StatusInternalServerError)
			return
		}

		totalSize := int64(0)
		for _, img := range images {
			totalSize += int64(len(img.ImageData))
		}

		processingTime := time.Since(startTime).Milliseconds()
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "pdf_to_images",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			InputSizeBytes:   sql.NullInt64{Int64: header.Size, Valid: true},
			OutputSizeBytes:  sql.NullInt64{Int64: totalSize, Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
			Status:           "success",
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"count":   len(images),
			"images":  images,
		})
	}
}
