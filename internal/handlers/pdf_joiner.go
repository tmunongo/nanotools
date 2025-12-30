package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/services"
	"github.com/tmunongo/nanotools/web/templates/tools"
)

func PDFJoinerPageHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.PDFJoinerPage().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func JoinPDFsHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Limit total request size (e.g., 50MB)
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			http.Error(w, "File too large or invalid", http.StatusBadRequest)
			return
		}

		files := r.MultipartForm.File["pdf"]
		if len(files) < 2 {
			http.Error(w, "Please upload at least 2 PDF files", http.StatusBadRequest)
			return
		}

		// Create a temp directory for this request to store uploaded files
		tmpDir, err := os.MkdirTemp("", "pdf-join-req-*")
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(tmpDir)

		var inputPaths []string
		var totalInputSize int64

		// Save uploaded files to temp dir
		for i, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to open file %s", fileHeader.Filename), http.StatusBadRequest)
				return
			}
			defer file.Close()

			tempFilePath := filepath.Join(tmpDir, fmt.Sprintf("input-%d.pdf", i))
			tempFile, err := os.Create(tempFilePath)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			defer tempFile.Close()

			if _, err := io.Copy(tempFile, file); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			tempFile.Close() // Close explicitly to ensure flush

			inputPaths = append(inputPaths, tempFilePath)
			totalInputSize += fileHeader.Size
		}

		// Join the PDFs
		joinedData, err := services.JoinPDFs(inputPaths)
		if err != nil {
			_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
				ToolName:         "pdf_joiner",
				IpAddress:        r.RemoteAddr,
				UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
				InputSizeBytes:   sql.NullInt64{Int64: totalInputSize, Valid: true},
				ProcessingTimeMs: sql.NullInt64{Int64: time.Since(startTime).Milliseconds(), Valid: true},
				Status:           "error",
				ErrorMessage:     sql.NullString{String: err.Error(), Valid: true},
			})

			http.Error(w, fmt.Sprintf("Join failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Success Log
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "pdf_joiner",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			InputSizeBytes:   sql.NullInt64{Int64: totalInputSize, Valid: true},
			OutputSizeBytes:  sql.NullInt64{Int64: int64(len(joinedData)), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: time.Since(startTime).Milliseconds(), Valid: true},
			Status:           "success",
		})

		// Reset cursor or just send bytes
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename=\"joined.pdf\"")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(joinedData)))

		if _, err := w.Write(joinedData); err != nil {
			// Cannot write http error here as we've already started writing response
			fmt.Printf("Error sending response: %v\n", err)
		}
	}
}
