package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type VideoDownloadOptions struct {
	URL                string
	Quality            string
	Format             string
	MaxDuration        int
	MaxFileSize        int64
	SubtitlesLang      string
	CookiesPath        string // optional path to cookies file to pass to yt-dlp
	CookiesFromBrowser string // optional browser name for --cookies-from-browser (e.g., chrome, firefox)
}

type VideoInfo struct {
	Title       string   `json:"title"`
	Duration    int      `json:"duration"`
	Description string   `json:"description"`
	Uploader    string   `json:"uploader"`
	Thumbnail   string   `json:"thumbnail"`
	Formats     []Format `json:"formats"`
	IsYouTube   bool     `json:"is_youtube"`
	FileSize    int64    `json:"filesize"`
}

type Format struct {
	FormatID   string `json:"format_id"`
	Extension  string `json:"ext"`
	Resolution string `json:"resolution"`
	FileSize   int64  `json:"filesize"`
	FPS        int    `json:"fps"`
	VideoCodec string `json:"vcodec"`
	AudioCodec string `json:"acodec"`
}

func GetVideoInfo(ctx context.Context, videoURL string, cookiesFromBrowser string, cookiesPath string) (*VideoInfo, error) {
	ytDlpPath, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, fmt.Errorf("yt-dlp not found: %w (install with: pip install yt-dlp)", err)
	}

	if !isValidURL(videoURL) {
		return nil, fmt.Errorf("invalid URL: %s", videoURL)
	}

	isYouTube := isYouTubeURL(videoURL)

	args := []string{
		"--dump-json",
		"--no-playlist",
		"--skip-download",
		"--no-check-certificate",
		videoURL,
	}

	// Allow yt-dlp to use a JS runtime (node/deno) if available to avoid missing formats
	// and enable passing cookies via YTDLP_COOKIES env var.
	// Move the URL out, so we can insert optional args before it.
	last := args[len(args)-1]
	args = args[:len(args)-1]

	// detect preferred JS runtime
	if _, err := exec.LookPath("node"); err == nil {
		args = append(args, "--js-runtimes", "node")
	} else if _, err := exec.LookPath("deno"); err == nil {
		args = append(args, "--js-runtimes", "deno")
	}

	// prefer explicit cookies file path, then cookies-from-browser, then env
	if cookiesPath != "" {
		args = append(args, "--cookies", cookiesPath)
	} else if cookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", cookiesFromBrowser)
	} else if cp := os.Getenv("YTDLP_COOKIES"); cp != "" {
		args = append(args, "--cookies", cp)
	}

	// put the URL back as last arg
	args = append(args, last)

	cmd := exec.CommandContext(ctx, ytDlpPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get video info: %w\nError: %s", err, stderr.String())
	}

	var info VideoInfo
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		return nil, fmt.Errorf("failed to parse video info: %w", err)
	}

	info.IsYouTube = isYouTube

	return &info, nil
}

func DownloadVideo(ctx context.Context, opts VideoDownloadOptions) (string, error) {
	ytDlpPath, err := exec.LookPath("yt-dlp")
	if err != nil {
		return "", fmt.Errorf("yt-dlp not found: %w (install with: pip install yt-dlp)", err)
	}

	if !isValidURL(opts.URL) {
		return "", fmt.Errorf("invalid URL: %s", opts.URL)
	}

	tmpDir, err := os.MkdirTemp("", "video-download-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	outputTemplate := filepath.Join(tmpDir, "%(title)s.%(ext)s")

	args := buildDownloadArgs(opts, outputTemplate)

	// extract the URL (last element) so we can insert runtime/cookies before it
	last := args[len(args)-1]
	args = args[:len(args)-1]

	if _, err := exec.LookPath("node"); err == nil {
		args = append(args, "--js-runtimes", "node")
	} else if _, err := exec.LookPath("deno"); err == nil {
		args = append(args, "--js-runtimes", "deno")
	}

	// cookies: prefer CookiesFromBrowser, then CookiesPath, then env
	if opts.CookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", opts.CookiesFromBrowser)
	} else {
		cookies := opts.CookiesPath
		if cookies == "" {
			cookies = os.Getenv("YTDLP_COOKIES")
		}
		if cookies != "" {
			args = append(args, "--cookies", cookies)
		}
	}

	args = append(args, last)

	cmd := exec.CommandContext(ctx, ytDlpPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("download failed: %w\nError: %s", err, stderr.String())
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to read download directory: %w", err)
	}

	if len(files) == 0 {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("no file was downloaded")
	}

	downloadedFile := filepath.Join(tmpDir, files[0].Name())

	if opts.MaxFileSize > 0 {
		fileInfo, err := os.Stat(downloadedFile)
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to check file size: %w", err)
		}

		if fileInfo.Size() > opts.MaxFileSize {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("file size (%d bytes) exceeds limit (%d bytes)",
				fileInfo.Size(), opts.MaxFileSize)
		}
	}

	return downloadedFile, nil
}

func buildDownloadArgs(opts VideoDownloadOptions, outputTemplate string) []string {
	args := []string{
		"--no-playlist",
		"--no-check-certificate",
		"-o", outputTemplate,
	}

	if opts.Quality == "audio" {
		args = append(args,
			"-f", "bestaudio",
			"--extract-audio",
			"--audio-format", "mp3",
			"--audio-quality", "0",
		)
	} else {
		formatSelector := buildFormatSelector(opts.Quality)
		args = append(args, "-f", formatSelector)

		args = append(args, "--merge-output-format", opts.Format)
	}

	if opts.SubtitlesLang != "" {
		args = append(args,
			"--write-subs",
			"--sub-lang", opts.SubtitlesLang,
			"--embed-subs",
		)
	}

	if opts.MaxDuration > 0 {
		args = append(args, "--match-filter",
			fmt.Sprintf("duration <= %d", opts.MaxDuration))
	}

	args = append(args, opts.URL)

	return args
}

func buildFormatSelector(quality string) string {
	switch quality {
	case "2160p", "4k":
		// 4K video with best audio
		return "bestvideo[height<=2160]+bestaudio/best[height<=2160]"
	case "1440p", "2k":
		// 1440p video with best audio
		return "bestvideo[height<=1440]+bestaudio/best[height<=1440]"
	case "1080p":
		// Full HD video with best audio
		return "bestvideo[height<=1080]+bestaudio/best[height<=1080]"
	case "720p":
		// HD video with best audio
		return "bestvideo[height<=720]+bestaudio/best[height<=720]"
	case "480p":
		// SD video with best audio
		return "bestvideo[height<=480]+bestaudio/best[height<=480]"
	case "360p":
		// Low quality for slow connections
		return "bestvideo[height<=360]+bestaudio/best[height<=360]"
	case "best":
		fallthrough
	default:
		// Best available quality
		return "bestvideo+bestaudio/best"
	}
}

func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func isYouTubeURL(urlStr string) bool {
	patterns := []string{
		`^https?://(?:www\.)?youtube\.com/watch\?v=`,
		`^https?://(?:www\.)?youtube\.com/embed/`,
		`^https?://youtu\.be/`,
		`^https?://(?:www\.)?youtube\.com/v/`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, urlStr)
		if matched {
			return true
		}
	}

	return false
}

func GetSupportedSites(ctx context.Context) ([]string, error) {
	ytDlpPath, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, fmt.Errorf("yt-dlp not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, ytDlpPath, "--list-extractors")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get supported sites: %w", err)
	}

	lines := strings.Split(stdout.String(), "\n")
	sites := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sites = append(sites, line)
		}
	}

	return sites, nil
}

func StreamingDownload(ctx context.Context, opts VideoDownloadOptions, w io.Writer) error {
	ytDlpPath, err := exec.LookPath("yt-dlp")
	if err != nil {
		return fmt.Errorf("yt-dlp not found: %w", err)
	}

	if !isValidURL(opts.URL) {
		return fmt.Errorf("invalid URL: %s", opts.URL)
	}

	args := []string{
		"--no-playlist",
		"--no-check-certificate",
		"-o", "-", // Output to stdout
	}

	formatSelector := buildFormatSelector(opts.Quality)
	args = append(args, "-f", formatSelector)

	args = append(args, opts.URL)
	last := args[len(args)-1]
	args = args[:len(args)-1]

	if _, err := exec.LookPath("node"); err == nil {
		args = append(args, "--js-runtimes", "node")
	} else if _, err := exec.LookPath("deno"); err == nil {
		args = append(args, "--js-runtimes", "deno")
	}

	// prefer CookiesFromBrowser, then CookiesPath, then env
	if opts.CookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", opts.CookiesFromBrowser)
	} else {
		cookies := opts.CookiesPath
		if cookies == "" {
			cookies = os.Getenv("YTDLP_COOKIES")
		}
		if cookies != "" {
			args = append(args, "--cookies", cookies)
		}
	}

	args = append(args, last)

	// Execute and pipe stdout directly to writer
	cmd := exec.CommandContext(ctx, ytDlpPath, args...)
	cmd.Stdout = w

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("streaming download failed: %w\nError: %s", err, stderr.String())
	}

	return nil
}

// EstimateDownloadTime estimates how long a download will take
// This is a rough estimate based on typical download speeds
func EstimateDownloadTime(fileSize int64, speedMbps float64) time.Duration {
	// Convert Mbps to bytes per second
	bytesPerSecond := (speedMbps * 1000000) / 8

	// Calculate seconds needed
	seconds := float64(fileSize) / bytesPerSecond

	return time.Duration(seconds) * time.Second
}

// CleanupDownloadedFile removes a downloaded file and its directory
// Call this after you've finished using the downloaded file
func CleanupDownloadedFile(filePath string) error {
	// Get the directory
	dir := filepath.Dir(filePath)

	// Remove the entire temporary directory
	return os.RemoveAll(dir)
}
