package logic

import (
	"fmt"
	"net/http"

	"github.com/minio/selfupdate"
)

// UpdateResult contains the result of an update operation
type UpdateResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Version string `json:"version"`
}

// PerformUpdate downloads the binary from the given URL and applies it
func PerformUpdate(url string, version string) (*UpdateResult, error) {
	fmt.Printf("Starting update to version %s from %s\n", version, url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		// Rollback is handled by library if possible, but mostly it's a replace
		return nil, fmt.Errorf("failed to apply update: %w", err)
	}

	return &UpdateResult{
		Success: true,
		Message: "Update applied successfully. Please restart the agent.",
		Version: version,
	}, nil
}
