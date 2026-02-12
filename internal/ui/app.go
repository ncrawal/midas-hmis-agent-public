package ui

import (
	"context"
	"fmt"
	"health-hmis-agent/internal/logic"
	"health-hmis-agent/internal/models"
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	logic.SetOnUpdateCallback(func(jobs []*models.PrintJob) {
		runtime.EventsEmit(a.ctx, "queue_updated", jobs)
	})
}

func (a *App) GetStatus() string {
	return fmt.Sprintf("Health HMIS Agent v%s is running on port %s", models.AgentVersion, models.DefaultPort)
}

func (a *App) GetFullInfo() map[string]interface{} {
	info := logic.GetDeviceInfo()
	printers, _ := logic.GetPrinters()

	return map[string]interface{}{
		"os":         info.OS,
		"mac":        info.MAC,
		"macs":       info.MACs,
		"ip":         info.IP,
		"hostname":   info.Hostname,
		"version":    models.AgentVersion,
		"port":       models.DefaultPort,
		"printers":   printers,
		"storageDir": logic.GetJobsDir(),
	}
}

func (a *App) GetInfo() map[string]string {
	return map[string]string{
		"purpose":      "Device Identification & Silent Print Agent for Midas Health HMIS",
		"security":     "No background mining, no hidden installs, restricted to localhost only.",
		"transparency": "This app only listens on 127.0.0.1:3033 for local requests from your authorized HMIS portal.",
	}
}

func (a *App) GetJobs() []*models.PrintJob {
	return logic.GetJobs()
}

func (a *App) SyncQueue() {
	logic.SyncQueue()
}

func (a *App) ClearJobs() {
	logic.ClearCompletedJobs()
}

func (a *App) DeleteJob(id string) {
	log.Printf("WAILS: DeleteJob called for ID: %s", id)
	logic.RemoveJob(id)
}

func (a *App) GetJobPDF(id string) (string, error) {
	return logic.GetJobPDFBase64(id)
}
