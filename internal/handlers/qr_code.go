package handlers

import (
	"database/sql"
	"fmt"
	"image/color"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/services"
	"github.com/tmunongo/nanotools/web/templates/tools"
)

// QRCodePageHandler serves the QR code generator page
func QRCodePageHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.QRCodePage().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// QRCodeGenerateHandler handles QR code generation requests
func QRCodeGenerateHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		if err := r.ParseMultipartForm(1 << 20); err != nil { // 1MB limit
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// Get common parameters
		qrType := r.FormValue("type")
		sizeStr := r.FormValue("size")
		errorCorrectionStr := r.FormValue("error_correction")
		fgColor := r.FormValue("foreground_color")
		bgColor := r.FormValue("background_color")

		// Parse size
		size, err := strconv.Atoi(sizeStr)
		if err != nil || size < 64 {
			size = 512
		}

		// Parse error correction
		errorCorrectionLevel, err := strconv.Atoi(errorCorrectionStr)
		if err != nil || errorCorrectionLevel < 0 || errorCorrectionLevel > 3 {
			errorCorrectionLevel = 1 // Medium
		}

		var qrData []byte
		var contentDescription string

		// Generate QR code based on type
		switch qrType {
		case "text":
			content := r.FormValue("content")
			if content == "" {
				http.Error(w, "Content is required", http.StatusBadRequest)
				return
			}

			qrData, err = services.GenerateQRCode(services.QRCodeOptions{
				Content:         content,
				Size:            size,
				ErrorCorrection: qrcode.RecoveryLevel(errorCorrectionLevel),
				ForegroundColor: parseColor(fgColor),
				BackgroundColor: parseColor(bgColor),
			})
			contentDescription = "text"

		case "wifi":
			ssid := r.FormValue("ssid")
			password := r.FormValue("password")
			encryption := r.FormValue("encryption")

			if ssid == "" {
				http.Error(w, "SSID is required", http.StatusBadRequest)
				return
			}

			qrData, err = services.GenerateWiFiQRCode(ssid, password, encryption, size)
			contentDescription = fmt.Sprintf("wifi:%s", ssid)

		case "vcard":
			name := r.FormValue("name")
			phone := r.FormValue("phone")
			email := r.FormValue("email")

			if name == "" {
				http.Error(w, "Name is required", http.StatusBadRequest)
				return
			}

			qrData, err = services.GenerateVCardQRCode(name, phone, email, size)
			contentDescription = fmt.Sprintf("vcard:%s", name)

		default:
			http.Error(w, "Invalid QR code type", http.StatusBadRequest)
			return
		}

		if err != nil {
			// Log error
			_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
				ToolName:         "qr_code_generator",
				IpAddress:        r.RemoteAddr,
				UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
				ProcessingTimeMs: sql.NullInt64{Int64: time.Since(startTime).Milliseconds(), Valid: true},
				Status:           "error",
				ErrorMessage:     sql.NullString{String: err.Error(), Valid: true},
			})

			http.Error(w, fmt.Sprintf("Failed to generate QR code: %v", err), http.StatusInternalServerError)
			return
		}

		// Log success
		processingTime := time.Since(startTime).Milliseconds()
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "qr_code_generator",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			InputSizeBytes:   sql.NullInt64{Int64: int64(len(contentDescription)), Valid: true},
			OutputSizeBytes:  sql.NullInt64{Int64: int64(len(qrData)), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
			Status:           "success",
		})

		// Return the QR code image
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", "inline; filename=\"qr-code.png\"")
		w.Header().Set("Content-Length", strconv.Itoa(len(qrData)))
		w.Write(qrData)
	}
}

// parseColor converts a hex color string to color.Color
// Returns nil if the string is invalid or empty
func parseColor(hexColor string) color.Color {
	hexColor = strings.TrimPrefix(hexColor, "#")
	if len(hexColor) != 6 {
		return nil
	}

	// Parse RGB components
	r, err1 := strconv.ParseUint(hexColor[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hexColor[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hexColor[4:6], 16, 8)

	if err1 != nil || err2 != nil || err3 != nil {
		return nil
	}

	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}
