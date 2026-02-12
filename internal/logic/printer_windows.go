//go:build windows

package logic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SilentPrint performs the silent printing on Windows using SumatraPDF (preferred) or PowerShell
func SilentPrint(filePath string, printerName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute) // Increased timeout for potential download
	defer cancel()

	// 1. Try SumatraPDF first (truly silent)
	sumatraPath, err := ensureSumatraPDF()
	if err == nil && sumatraPath != "" {
		var args []string
		if printerName != "" {
			args = append(args, "-print-to", printerName)
		} else {
			args = append(args, "-print-to-default")
		}
		args = append(args, "-silent", filePath)

		cmd := exec.CommandContext(ctx, sumatraPath, args...)
		if output, err := cmd.CombinedOutput(); err == nil {
			return "", nil
		} else {
			// Log error but try fallback
			fmt.Printf("SumatraPDF failed: %v, output: %s. Trying PowerShell fallback...\n", err, string(output))
		}
	} else {
		fmt.Printf("SumatraPDF not available: %v. Using PowerShell fallback...\n", err)
	}

	// 2. Fallback to PowerShell (might show dialog/window briefly)
	var cmd *exec.Cmd
	if printerName != "" {
		script := fmt.Sprintf(`Start-Process -FilePath "%s" -Verb PrintTo -ArgumentList "%s" -WindowStyle Hidden -Wait`, filePath, printerName)
		cmd = exec.CommandContext(ctx, "powershell", "-Command", script)
	} else {
		script := fmt.Sprintf(`Start-Process -FilePath "%s" -Verb Print -WindowStyle Hidden -Wait`, filePath)
		cmd = exec.CommandContext(ctx, "powershell", "-Command", script)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("windows printing timed out after 2 minutes")
		}
		return "", fmt.Errorf("windows print failed: %v, output: %s", err, string(output))
	}
	return "", nil
}

// ensureSumatraPDF checks for SumatraPDF.exe and downloads it if missing
func ensureSumatraPDF() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exePath)
	sumatraPath := filepath.Join(dir, "SumatraPDF.exe")

	if _, err := os.Stat(sumatraPath); err == nil {
		return sumatraPath, nil
	}

	// Download it
	fmt.Println("Downloading SumatraPDF for silent printing...")
	url := "https://github.com/sumatrapdfreader/sumatrapdf/releases/download/3.5.2/SumatraPDF-3.5.2-64.exe"
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download SumatraPDF: %v", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(sumatraPath)
	if err != nil {
		return "", fmt.Errorf("failed to create SumatraPDF file: %v", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write SumatraPDF file: %v", err)
	}

	return sumatraPath, nil
}

// GetPrinters returns a list of available printer names on Windows
func GetPrinters() ([]string, error) {
	cmd := exec.Command("powershell", "-Command", "Get-Printer | Select-Object -ExpandProperty Name")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var printers []string
	for _, l := range lines {
		if s := strings.TrimSpace(l); s != "" {
			printers = append(printers, s)
		}
	}
	return printers, nil
}

// CheckOSQueue is a placeholder
func CheckOSQueue() (map[string]bool, error) {
	return nil, nil // Windows monitoring not implemented yet
}

// CancelOSJob is a placeholder
func CancelOSJob(osJobID string) error {
	return nil
}
