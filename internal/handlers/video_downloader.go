package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tmunongo/nanotools/internal/db"
	"github.com/tmunongo/nanotools/internal/services"
	"github.com/tmunongo/nanotools/web/templates/tools"
)

// detectBrowserFromUA maps a User-Agent string to a yt-dlp browser name
func detectBrowserFromUA(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "firefox"):
		return "firefox"
	case strings.Contains(ua, "edg") || strings.Contains(ua, "edge"):
		return "edge"
	case strings.Contains(ua, "opr") || strings.Contains(ua, "opera"):
		return "opera"
	case strings.Contains(ua, "brave"):
		return "brave"
	case strings.Contains(ua, "chromium"):
		return "chromium"
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		return "safari"
	case strings.Contains(ua, "chrome"):
		return "chrome"
	default:
		return ""
	}
}

func VideoDownloaderPageHandler(w http.ResponseWriter, r *http.Request) {
	err := tools.VideoDownloaderPage().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func VideoInfoHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		contentTypeHeader := r.Header.Get("Content-Type")
		var uploadedCookiesPath string
		if strings.HasPrefix(contentTypeHeader, "multipart/form-data") {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, "Invalid multipart form data", http.StatusBadRequest)
				return
			}

			cf, _, ferr := r.FormFile("cookies_file")
			if ferr == nil && cf != nil {
				defer cf.Close()
				tmp, terr := os.CreateTemp("", "cookies-*.txt")
				if terr == nil {
					defer tmp.Close()
					io.Copy(tmp, cf)
					uploadedCookiesPath = tmp.Name()
					os.Chmod(uploadedCookiesPath, 0600)
				}
			}
		} else {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Invalid form data", http.StatusBadRequest)
				return
			}
		}

		videoURL := r.FormValue("url")
		if videoURL == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		// temporarily disable YouTube support
		if services.IsYouTubeURL(videoURL) {
			http.Error(w, "YouTube video info is temporarily disabled", http.StatusForbidden)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// detect browser from user agent and pass it to yt-dlp so it can use
		// --cookies-from-browser if needed. Prefer uploaded cookies file.
		browser := detectBrowserFromUA(r.UserAgent())
		info, err := services.GetVideoInfo(ctx, videoURL, browser, uploadedCookiesPath)
		if err != nil {
			_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
				ToolName:         "video_downloader_info",
				IpAddress:        r.RemoteAddr,
				UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
				ProcessingTimeMs: sql.NullInt64{Int64: time.Since(startTime).Milliseconds(), Valid: true},
				Status:           "error",
				ErrorMessage:     sql.NullString{String: err.Error(), Valid: true},
			})

			http.Error(w, fmt.Sprintf("Failed to get video info: %v", err), http.StatusBadRequest)
			return
		}

		processingTime := time.Since(startTime).Milliseconds()
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "video_downloader_info",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
			Status:           "success",
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}
}

func VideoDownloadHandler(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Support both application/x-www-form-urlencoded and multipart/form-data
		contentTypeHeader := r.Header.Get("Content-Type")
		if strings.HasPrefix(contentTypeHeader, "multipart/form-data") {
			if err := r.ParseMultipartForm(50 << 20); err != nil {
				http.Error(w, "Invalid multipart form data", http.StatusBadRequest)
				return
			}
		} else {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Invalid form data", http.StatusBadRequest)
				return
			}
		}
		var uploadedCookiesPath string

		// check for an uploaded cookies file
		cookiesFile, _, ferr := r.FormFile("cookies_file")
		if ferr == nil && cookiesFile != nil {
			defer cookiesFile.Close()
			tmp, terr := os.CreateTemp("", "cookies-*.txt")
			if terr == nil {
				defer tmp.Close()
				io.Copy(tmp, cookiesFile)
				uploadedCookiesPath = tmp.Name()
				// ensure the temp file is readable by yt-dlp
				os.Chmod(uploadedCookiesPath, 0600)
			}
		}

		videoURL := r.FormValue("url")
		quality := r.FormValue("quality")
		format := r.FormValue("format")
		subtitlesLang := r.FormValue("subtitles")

		if videoURL == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		// Temporarily disable YouTube downloads
		if services.IsYouTubeURL(videoURL) {
			http.Error(w, "YouTube downloads are temporarily disabled", http.StatusForbidden)
			return
		}

		if quality == "" {
			quality = "720p"
		}
		if format == "" {
			format = "mp4"
		}

		// large videos can take time
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()

		browser := detectBrowserFromUA(r.UserAgent())

		filePath, err := services.DownloadVideo(ctx, services.VideoDownloadOptions{
			URL:                videoURL,
			Quality:            quality,
			Format:             format,
			SubtitlesLang:      subtitlesLang,
			MaxFileSize:        500 * 1024 * 1024,
			MaxDuration:        3600,
			CookiesFromBrowser: browser,
			CookiesPath:        uploadedCookiesPath,
		})

		if err != nil {
			_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
				ToolName:         "video_downloader",
				IpAddress:        r.RemoteAddr,
				UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
				ProcessingTimeMs: sql.NullInt64{Int64: time.Since(startTime).Milliseconds(), Valid: true},
				Status:           "error",
				ErrorMessage:     sql.NullString{String: err.Error(), Valid: true},
			})

			http.Error(w, fmt.Sprintf("Download failed: %v", err), http.StatusInternalServerError)
			return
		}

		defer services.CleanupDownloadedFile(filePath)

		file, err := os.Open(filePath)
		if err != nil {
			http.Error(w, "Failed to open downloaded file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			http.Error(w, "Failed to get file info", http.StatusInternalServerError)
			return
		}

		processingTime := time.Since(startTime).Milliseconds()
		_, _ = queries.CreateAuditLog(r.Context(), db.CreateAuditLogParams{
			ToolName:         "video_downloader",
			IpAddress:        r.RemoteAddr,
			UserAgent:        sql.NullString{String: r.UserAgent(), Valid: true},
			OutputSizeBytes:  sql.NullInt64{Int64: fileInfo.Size(), Valid: true},
			ProcessingTimeMs: sql.NullInt64{Int64: processingTime, Valid: true},
			Status:           "success",
		})

		contentType := "video/mp4"
		if quality == "audio" {
			contentType = "audio/mpeg"
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileInfo.Name()))
		w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

		_, err = io.Copy(w, file)
		if err != nil {
			fmt.Printf("Error streaming file: %v\n", err)
		}
	}
}
