package services

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"github.com/chai2010/webp"
)

type ImageConvertOptions struct {
	OutputFormat string
	Quality      int
}

// use streaming to avoid loading the entire image into memory multiple times
func ConvertImage(input io.Reader, output io.Writer, opts ImageConvertOptions) error {
	img, format, err := image.Decode(input)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// validate we support the input format
	supportedInputs := []string{"jpeg", "png", "webp"}
	if !contains(supportedInputs, strings.ToLower(format)) {
		return fmt.Errorf("unsupported input format: %s", format)
	}

	if opts.Quality < 1 || opts.Quality > 100 {
		opts.Quality = 85
	}

	switch strings.ToLower(opts.OutputFormat) {
	case "jpeg", "jpg":
		return jpeg.Encode(output, img, &jpeg.Options{
			Quality: opts.Quality,
		})

	case "png":
		encoder := png.Encoder{
			CompressionLevel: png.DefaultCompression,
		}
		return encoder.Encode(output, img)

	case "webp":
		return webp.Encode(output, img, &webp.Options{
			Quality: float32(opts.Quality),
		})

	default:
		return fmt.Errorf("unsupported output format: %s", opts.OutputFormat)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
