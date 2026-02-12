package logic

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ConvertHTMLToPDF converts HTML string to PDF file and returns the file path
func ConvertHTMLToPDF(html string) (string, error) {
	// Create temp HTML file
	tmpHTML, err := os.CreateTemp("", "bill_*.html")
	if err != nil {
		return "", fmt.Errorf("failed to create temp HTML file: %w", err)
	}
	htmlPath := tmpHTML.Name()
	defer os.Remove(htmlPath) // Clean up HTML after conversion

	// Write HTML content
	if _, err := tmpHTML.WriteString(html); err != nil {
		tmpHTML.Close()
		return "", fmt.Errorf("failed to write HTML: %w", err)
	}
	tmpHTML.Close()

	// Create temp PDF file
	tmpPDF, err := os.CreateTemp("", "bill_*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp PDF file: %w", err)
	}
	pdfPath := tmpPDF.Name()
	tmpPDF.Close()

	// Convert HTML to PDF using wkhtmltopdf or chrome headless
	if err := convertWithChrome(htmlPath, pdfPath); err != nil {
		// Fallback to wkhtmltopdf if chrome fails
		if err2 := convertWithWkhtmltopdf(htmlPath, pdfPath); err2 != nil {
			os.Remove(pdfPath)
			return "", fmt.Errorf("PDF conversion failed (chrome: %v, wkhtmltopdf: %v)", err, err2)
		}
	}

	return pdfPath, nil
}

// convertWithChrome uses Chrome/Chromium headless to convert HTML to PDF
func convertWithChrome(htmlPath, pdfPath string) error {
	var chromePath string

	switch runtime.GOOS {
	case "darwin":
		chromePath = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	case "linux":
		// Try common Linux paths
		for _, path := range []string{
			"/usr/bin/google-chrome",
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
		} {
			if _, err := os.Stat(path); err == nil {
				chromePath = path
				break
			}
		}
	case "windows":
		chromePath = "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
	}

	if chromePath == "" || !fileExists(chromePath) {
		return fmt.Errorf("chrome not found")
	}

	absHTMLPath, _ := filepath.Abs(htmlPath)
	absPDFPath, _ := filepath.Abs(pdfPath)

	cmd := exec.Command(chromePath,
		"--headless",
		"--disable-gpu",
		"--print-to-pdf="+absPDFPath,
		"file://"+absHTMLPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chrome conversion failed: %w", err)
	}

	return nil
}

// convertWithWkhtmltopdf uses wkhtmltopdf to convert HTML to PDF
func convertWithWkhtmltopdf(htmlPath, pdfPath string) error {
	cmd := exec.Command("wkhtmltopdf", htmlPath, pdfPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wkhtmltopdf conversion failed: %w", err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
