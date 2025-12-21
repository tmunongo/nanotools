package services

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/chai2010/webp"
)

type PDFToImagesOptions struct {
	DPI       int
	Format    string
	Quality   int
	FirstPage int
	LastPage  int
}

type PDFPageImage struct {
	PageNumber int
	ImageData  []byte
	Format     string
}

func ConvertPDFToImages(pdfReader io.Reader, opts PDFToImagesOptions) ([]PDFPageImage, error) {
	if opts.DPI < 72 || opts.DPI > 600 {
		opts.DPI = 150
	}
	if opts.Quality < 1 || opts.Quality > 100 {
		opts.Quality = 85
	}

	gsPath, err := exec.LookPath("gs")
	if err != nil {
		return nil, fmt.Errorf("Ghostscript not found: %w (install with: apt-get install ghostscript)", err)
	}

	tmpDir, err := os.MkdirTemp("", "pdf-convert-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	pdfFile, err := os.Create(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp PDF: %w", err)
	}

	_, err = io.Copy(pdfFile, pdfReader)
	pdfFile.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to write PDF: %w", err)
	}

	device := "png16m"
	switch opts.Format {
	case "jpeg", "jpg":
		device = "jpeg"
	case "png":
		device = "png16m"
	case "webp":
		device = "png16m"
	}

	outputPattern := filepath.Join(tmpDir, "page-%04d."+opts.Format)

	args := []string{
		"-dNOPAUSE",
		"-dBATCH",
		"-dSAFER",
		"-sDEVICE=" + device,
		"-r" + strconv.Itoa(opts.DPI),
		"-sOutputFile=" + outputPattern,
		pdfPath,
	}

	if opts.Format == "jpeg" {
		args = append(args[:6], append([]string{
			"-dJPEGQ=" + strconv.Itoa(opts.Quality),
		}, args[6:]...)...)
	}

	if opts.FirstPage > 0 {
		pageRange := fmt.Sprintf("-dFirstPage=%d", opts.FirstPage)
		args = append([]string{args[0]}, append([]string{pageRange}, args[1:]...)...)
	}
	if opts.LastPage > 0 {
		pageRange := fmt.Sprintf("-dLastPage=%d", opts.LastPage)
		args = append([]string{args[0]}, append([]string{pageRange}, args[1:]...)...)
	}

	cmd := exec.Command(gsPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ghostscript failed: %w\nOutput: %s", err, string(output))
	}

	images, err := loadGeneratedImages(tmpDir, opts.Format)
	if err != nil {
		return nil, fmt.Errorf("failed to load images: %w", err)
	}

	if opts.Format == "webp" {
		images, err = convertImagesToWebP(images, opts.Quality)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to WebP: %w", err)
		}
	}

	return images, nil
}

func loadGeneratedImages(dir string, format string) ([]PDFPageImage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var images []PDFPageImage
	pageNum := 1

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) == "."+format ||
			(format == "webp" && filepath.Ext(entry.Name()) == ".png") {

			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}

			images = append(images, PDFPageImage{
				PageNumber: pageNum,
				ImageData:  data,
				Format:     format,
			})
			pageNum++
		}
	}

	return images, nil
}

func convertImagesToWebP(images []PDFPageImage, quality int) ([]PDFPageImage, error) {
	var converted []PDFPageImage

	for _, img := range images {
		imgReader := bytes.NewReader(img.ImageData)
		decodedImg, err := png.Decode(imgReader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode PNG: %w", err)
		}

		var buf bytes.Buffer
		err = webp.Encode(&buf, decodedImg, &webp.Options{
			Quality: float32(quality),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to encode WebP: %w", err)
		}

		converted = append(converted, PDFPageImage{
			PageNumber: img.PageNumber,
			ImageData:  buf.Bytes(),
			Format:     "webp",
		})
	}

	return converted, nil
}
