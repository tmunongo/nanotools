package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// JoinPDFs takes a list of paths to PDF files and joins them into a single PDF.
// It returns the byte content of the joined PDF.
func JoinPDFs(inputPaths []string) ([]byte, error) {
	if len(inputPaths) == 0 {
		return nil, fmt.Errorf("no input files provided")
	}

	gsPath, err := exec.LookPath("gs")
	if err != nil {
		return nil, fmt.Errorf("Ghostscript not found: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "pdf-join-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "output.pdf")

	// ghostscript -dNOPAUSE -sDEVICE=pdfwrite -sOUTPUTFILE=output.pdf -dBATCH input1.pdf input2.pdf
	args := []string{
		"-dNOPAUSE",
		"-sDEVICE=pdfwrite",
		"-sOUTPUTFILE=" + outputPath,
		"-dBATCH",
	}
	args = append(args, inputPaths...)

	cmd := exec.Command(gsPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ghostscript failed: %w\nOutput: %s", err, string(output))
	}

	// Read the output file
	joinedPDF, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output PDF: %w", err)
	}

	return joinedPDF, nil
}
