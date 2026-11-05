package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// UnlockPDF removes the encryption from a password-protected PDF.
// The password is used only for decryption and is never logged or persisted.
// Returns the byte content of the unlocked PDF.
func UnlockPDF(inputPath, password string) ([]byte, error) {
	qpdfPath, err := exec.LookPath("qpdf")
	if err != nil {
		// Fallback to Ghostscript if qpdf is not available
		return unlockWithGhostscript(inputPath, password)
	}
	return unlockWithQPDF(qpdfPath, inputPath, password)
}

func unlockWithQPDF(qpdfPath, inputPath, password string) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "pdf-unlock-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "unlocked.pdf")

	// qpdf --password=<pw> --decrypt <input> <output>
	// Note: we pass the password via flag, not shell interpolation, so it is
	// never written to disk or echoed into a shell string.
	cmd := exec.Command(qpdfPath,
		"--password="+password,
		"--decrypt",
		inputPath,
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Mask the password from any error output before surfacing it.
		errMsg := strings.ReplaceAll(string(output), password, "[REDACTED]")
		// qpdf exit code 2 means wrong password / bad input.
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 2 {
			return nil, fmt.Errorf("incorrect password or the PDF is not encrypted")
		}
		return nil, fmt.Errorf("qpdf failed: %s", errMsg)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read unlocked PDF: %w", err)
	}
	return data, nil
}

func unlockWithGhostscript(inputPath, password string) ([]byte, error) {
	gsPath, err := exec.LookPath("gs")
	if err != nil {
		return nil, fmt.Errorf("neither qpdf nor Ghostscript found on this server")
	}

	tmpDir, err := os.MkdirTemp("", "pdf-unlock-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "unlocked.pdf")

	// gs -dNOPAUSE -sDEVICE=pdfwrite -dBATCH -sPDFPassword=<pw> -sOUTPUTFILE=<out> <in>
	cmd := exec.Command(gsPath,
		"-dNOPAUSE",
		"-sDEVICE=pdfwrite",
		"-dBATCH",
		"-sPDFPassword="+password,
		"-sOUTPUTFILE="+outputPath,
		inputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.ReplaceAll(string(output), password, "[REDACTED]")
		return nil, fmt.Errorf("ghostscript failed: %s", errMsg)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read unlocked PDF: %w", err)
	}
	return data, nil
}
