package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/services"
	"github.com/tmunongo/nanotools/web/templates/tools"
)

func PDFUnlockerPageHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.PDFUnlockerPage().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func UnlockPDFHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Limit request size to 50 MB
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			http.Error(w, "File too large or invalid form data", http.StatusBadRequest)
			return
		}

		// --- File ---
		fileHeader, _, err := r.FormFile("pdf")
		if err != nil {
			http.Error(w, "Please upload a PDF file", http.StatusBadRequest)
			return
		}
		defer fileHeader.Close()

		// Retrieve the original file name from the multipart header
		multipartHeaders := r.MultipartForm.File["pdf"]
		var originalName string
		if len(multipartHeaders) > 0 {
			originalName = multipartHeaders[0].Filename
		}
		if originalName == "" {
			originalName = "document.pdf"
		}

		// --- Password ---
		// Read once into a local variable; never log or write it anywhere.
		password := r.FormValue("password")
		if password == "" {
			http.Error(w, "Please provide the PDF password", http.StatusBadRequest)
			return
		}

		// Save upload to a temp file
		tmpDir, err := os.MkdirTemp("", "pdf-unlock-req-*")
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(tmpDir)

		inputPath := filepath.Join(tmpDir, "input.pdf")
		tmpFile, err := os.Create(inputPath)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		inputSize, err := io.Copy(tmpFile, fileHeader)
		tmpFile.Close()
		if err != nil {
			http.Error(w, "Failed to read uploaded file", http.StatusInternalServerError)
			return
		}

		// Unlock
		unlockedData, err := services.UnlockPDF(inputPath, password)
		if err != nil {
			_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
				ToolName:         "pdf_unlocker",
				IpAddress:        r.RemoteAddr,
				UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
				InputSizeBytes:   sql.NullInt64{Int64: inputSize, Valid: true},
				ProcessingTimeMs: sql.NullInt64{Int64: time.Since(startTime).Milliseconds(), Valid: true},
				Status:           "error",
				// Use a generic message — never surface the password in logs.
				ErrorMessage: sql.NullString{String: "unlock failed", Valid: true},
			})

			// Return a user-friendly message; don't forward raw tool stderr.
			msg := err.Error()
			if strings.Contains(msg, "incorrect password") || strings.Contains(msg, "not encrypted") {
				http.Error(w, "Incorrect password or the file is not password-protected", http.StatusUnprocessableEntity)
			} else {
				http.Error(w, "Failed to unlock PDF. Please check the file and password.", http.StatusInternalServerError)
			}
			return
		}

		// Success audit log — password is intentionally absent
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "pdf_unlocker",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			InputSizeBytes:   sql.NullInt64{Int64: inputSize, Valid: true},
			OutputSizeBytes:  sql.NullInt64{Int64: int64(len(unlockedData)), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: time.Since(startTime).Milliseconds(), Valid: true},
			Status:           "success",
		})

		// Build a safe download filename: "unlocked-<original>"
		baseName := filepath.Base(originalName)
		downloadName := fmt.Sprintf("unlocked-%s", baseName)

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", downloadName))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(unlockedData)))

		if _, err := w.Write(unlockedData); err != nil {
			fmt.Printf("Error sending unlocked PDF response: %v\n", err)
		}
	}
}
