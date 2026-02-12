package logic

import (
	"encoding/base64"
	"fmt"
	"health-hmis-agent/internal/models"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// PreviewFile opens the file with the default OS application
func PreviewFile(filePath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Powerfull quoting for PowerShell
		cmd = exec.Command("powershell", "-Command", "Start-Process", fmt.Sprintf("'%s'", filePath))
	case "darwin":
		cmd = exec.Command("open", filePath)
	default: // linux, etc.
		cmd = exec.Command("xdg-open", filePath)
	}
	return cmd.Start()
}

// SaveBase64ToFile decodes base64 data and saves it to a local temporary path
func SaveBase64ToFile(base64Data string) (string, error) {
	// Handle data URL prefix if present
	if idx := strings.Index(base64Data, ","); idx != -1 {
		base64Data = base64Data[idx+1:]
	}

	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "agent-print-*.pdf")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

// DownloadFile downloads a file from a URL to a local temporary path
func DownloadFile(url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set a descriptive User-Agent
	req.Header.Set("User-Agent", "Health-HMIS-Agent/"+models.AgentVersion)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "agent-print-*.pdf")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}
