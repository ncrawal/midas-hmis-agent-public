//go:build darwin || linux

package logic

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SilentPrint performs the silent printing on Unix systems
func SilentPrint(filePath string, printerName string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("cannot print: file path is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	args := []string{}
	if printerName != "" {
		args = append(args, "-d", printerName)
	}
	args = append(args, filePath)

	cmd := exec.CommandContext(ctx, "lp", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("printing timed out after 1 minute")
		}
		return "", fmt.Errorf("lp failed: %v, output: %s", err, string(output))
	}

	// Parse request id: "request id is Brother_xxx-123 (1 file(s))"
	outStr := string(output)
	if strings.Contains(outStr, "request id is") {
		parts := strings.Split(outStr, "is")
		if len(parts) > 1 {
			idPart := strings.TrimSpace(parts[1])
			idFields := strings.Fields(idPart)
			if len(idFields) > 0 {
				return idFields[0], nil
			}
		}
	}

	return "", nil
}

// GetPrinters returns a list of available printer names on Unix
func GetPrinters() ([]string, error) {
	cmd := exec.Command("lpstat", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var printers []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) > 0 {
			printers = append(printers, parts[0])
		}
	}
	return printers, nil
}

// CheckOSQueue returns a map of currently active Job IDs in the OS spooler
func CheckOSQueue() (map[string]bool, error) {
	cmd := exec.Command("lpstat", "-o")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	activeJobs := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Example line: "Brother_xxx-123 apple 1024 ..."
		fields := strings.Fields(line)
		if len(fields) > 0 {
			activeJobs[fields[0]] = true
		}
	}
	return activeJobs, nil
}

// CancelOSJob removes a job from the Unix print spooler
func CancelOSJob(osJobID string) error {
	if osJobID == "" {
		return nil
	}
	// Use 'cancel' command to remove job from spooler
	cmd := exec.Command("cancel", osJobID)
	return cmd.Run()
}
