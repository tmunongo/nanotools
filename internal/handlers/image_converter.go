package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/services"
	"github.com/tmunongo/nanotools/web/templates/tools"
)

func ImageConverterPageHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.ImageConverterPage().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func ImageConvertHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// 10MB limit
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "File too large or invalid", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "No image uploaded", http.StatusBadRequest)
			return
		}
		defer file.Close()

		outputFormat := r.FormValue("format")
		qualityStr := r.FormValue("quality")

		quality, err := strconv.Atoi(qualityStr)
		if err != nil {
			quality = 85
		}

		// for very large images better to use a temp file
		var outputBuffer bytes.Buffer

		err = services.ConvertImage(file, &outputBuffer, services.ImageConvertOptions{
			OutputFormat: outputFormat,
			Quality:      quality,
		})

		if err != nil {
			_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
				ToolName:         "image_converter",
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

		processingTime := time.Since(startTime).Milliseconds()
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "image_converter",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			InputSizeBytes:   sql.NullInt64{Int64: header.Size, Valid: true},
			OutputSizeBytes:  sql.NullInt64{Int64: int64(outputBuffer.Len()), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
			Status:           "success",
		})

		contentType := ""
		switch outputFormat {
		case "jpeg", "jpg":
			contentType = "image/jpeg"
		case "png":
			contentType = "image/png"
		case "webp":
			contentType = "image/webp"
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"converted.%s\"", outputFormat))
		w.Header().Set("Content-Length", strconv.Itoa(outputBuffer.Len()))

		_, err = w.Write(outputBuffer.Bytes())
		if err != nil {
			// can't return an error here as headers are already sent
			fmt.Printf("Error writing response: %v\n", err)
		}
	}
}
