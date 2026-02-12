package models

// DeviceInfo represents the core identity of the hardware.
type DeviceInfo struct {
	MAC      string   `json:"mac"`      // Primary/Best guess MAC
	MACs     []string `json:"mac_list"` // All valid MACs found
	IP       string   `json:"local_ip"`
	Hostname string   `json:"hostname"`
	OS       string   `json:"os_platform"`
	Version  string   `json:"agent_version"`
}

// PrintRequest holds the data for the printing command
type PrintRequest struct {
	FileURL    string `json:"fileUrl"`
	Base64     string `json:"base64,omitempty"`
	HTML       string `json:"html,omitempty"` // HTML content to convert to PDF
	Printer    string `json:"printer,omitempty"`
	Copies     int    `json:"copies,omitempty"`
	Preview    bool   `json:"preview,omitempty"`
	HospitalNo string `json:"hospitalNo,omitempty"`
	UserName   string `json:"userName,omitempty"`
}

// PrintJob represents a job in the queue
type PrintJob struct {
	ID         string `json:"id"`
	FileName   string `json:"fileName"`
	HospitalNo string `json:"hospitalNo"`
	UserName   string `json:"userName"`
	Printer    string `json:"printer"`
	Status     string `json:"status"` // pending, printing, queued, completed, failed
	OSJobID    string `json:"osJobId,omitempty"`
	Error      string `json:"error,omitempty"`
	CreatedAt  string `json:"createdAt"`
	FilePath   string `json:"filePath,omitempty"`
}

const (
	AgentVersion = "1.3.0"
	DefaultPort  = "3033"

	StatusPending   = "pending"
	StatusPrinting  = "printing"
	StatusQueued    = "queued"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)
