package services

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/skip2/go-qrcode"
)

type QRCodeOptions struct {
	Content string

	// dimension in pixels (QR codes are square)
	// Common values: 256, 512, 1024
	Size int

	// ErrorCorrection level: Low, Medium, High, Highest
	ErrorCorrection qrcode.RecoveryLevel

	ForegroundColor color.Color

	BackgroundColor color.Color
}

func GenerateQRCode(opts QRCodeOptions) ([]byte, error) {
	if opts.Content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	if opts.Size < 64 {
		opts.Size = 256
	}
	if opts.Size > 2048 {
		opts.Size = 2048 // Prevent enormous images
	}

	if opts.ErrorCorrection == 0 {
		// Medium is a good balance for most use cases
		opts.ErrorCorrection = qrcode.Medium
	}

	qr, err := qrcode.New(opts.Content, opts.ErrorCorrection)
	if err != nil {
		return nil, fmt.Errorf("failed to create QR code: %w", err)
	}

	if opts.ForegroundColor != nil || opts.BackgroundColor != nil {
		img := qr.Image(opts.Size)

		if opts.ForegroundColor != nil || opts.BackgroundColor != nil {
			img = applyCustomColors(img, opts.ForegroundColor, opts.BackgroundColor)
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("failed to encode PNG: %w", err)
		}

		return buf.Bytes(), nil
	}

	// Use the built-in PNG method for default black and white
	return qr.PNG(opts.Size)
}

func applyCustomColors(img image.Image, fg, bg color.Color) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	if fg == nil {
		fg = color.Black
	}
	if bg == nil {
		bg = color.White
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)

			r, g, b, _ := originalColor.RGBA()
			luminance := (r + g + b) / 3

			if luminance < 32768 { // Threshold for "dark"
				newImg.Set(x, y, fg)
			} else {
				newImg.Set(x, y, bg)
			}
		}
	}

	return newImg
}

// GenerateWiFiQRCode creates a QR code for Wi-Fi credentials
func GenerateWiFiQRCode(ssid, password, encryption string, size int) ([]byte, error) {
	// Validate encryption type
	validEncryption := map[string]bool{
		"WPA":    true,
		"WEP":    true,
		"":       true,
		"nopass": true,
	}

	if !validEncryption[encryption] {
		encryption = "WPA"
	}

	// Format: WIFI:T:WPA;S:network_name;P:password;;
	content := fmt.Sprintf("WIFI:T:%s;S:%s;P:%s;;", encryption, ssid, password)

	return GenerateQRCode(QRCodeOptions{
		Content:         content,
		Size:            size,
		ErrorCorrection: qrcode.High,
	})
}

// vCard is the standard format for contact info
func GenerateVCardQRCode(name, phone, email string, size int) ([]byte, error) {
	// Build a simple vCard (version 3.0)
	// vCard has a specific format that contact apps understand
	content := fmt.Sprintf(`BEGIN:VCARD
		VERSION:3.0
		FN:%s
		TEL:%s
		EMAIL:%s
		END:VCARD`, name, phone, email)

	return GenerateQRCode(QRCodeOptions{
		Content:         content,
		Size:            size,
		ErrorCorrection: qrcode.Medium,
	})
}

// func EmbedLogo(qrImage image.Image, logoImage image.Image) image.Image {
//     bounds := qrImage.Bounds()
//     result := image.NewRGBA(bounds)

//     // Draw the QR code
//     draw.Draw(result, bounds, qrImage, image.Point{}, draw.Src)

//     logoSize := bounds.Dx() / 5
//     logoPos := image.Rect(
//         (bounds.Dx()-logoSize)/2,
//         (bounds.Dy()-logoSize)/2,
//         (bounds.Dx()+logoSize)/2,
//         (bounds.Dy()+logoSize)/2,
//     )

//     draw.Draw(result, logoPos, logoImage, image.Point{}, draw.Over)

//     return result
// }
