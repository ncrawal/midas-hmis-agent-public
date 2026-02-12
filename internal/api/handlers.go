package api

import (
	"encoding/json"
	"fmt"
	"health-hmis-agent/internal/logic"
	"health-hmis-agent/internal/models"
	"log"
	"net/http"
	"os"
)

func RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/health-agent", handleDeviceInfo)
	mux.HandleFunc("/print", handlePrint)
	mux.HandleFunc("/printers", handlePrinters)
	mux.HandleFunc("/status", handleStatus)
	mux.HandleFunc("/queue", handleQueue)
	mux.HandleFunc("/queue/clear", handleClearQueue)
	mux.HandleFunc("/queue/retry", handleRetryJob)
	mux.HandleFunc("/queue/view", handleViewJob)
	mux.HandleFunc("/queue/delete", handleDeleteJob)
	mux.HandleFunc("/update", handleUpdate)
	mux.HandleFunc("/ws", handleWebSocket)

	// Set callback for logic updates (thread-safe setup)
	logic.SetOnUpdateCallback(func(jobs []*models.PrintJob) {
		BroadcastQueueUpdate(jobs)
	})
}

func handleDeviceInfo(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}
	info := logic.GetDeviceInfo()
	json.NewEncoder(w).Encode(info)
}

func handlePrint(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.PrintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FileURL == "" && req.Base64 == "" && req.HTML == "" {
		http.Error(w, "fileUrl, base64, or html is required", http.StatusBadRequest)
		return
	}

	job := logic.AddJob(req.Printer, req.FileURL, req.HospitalNo, req.UserName, "")
	if req.Base64 != "" && req.FileURL == "" {
		job.FileName = "base64_data.pdf"
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Job received and added to queue",
		"jobId":   job.ID,
	})

	go func(r models.PrintRequest, jobId string) {
		var localPath string
		var err error

		logic.UpdateJobStatus(jobId, models.StatusPending, "Preparing file...")

		if r.HTML != "" {
			localPath, err = logic.ConvertHTMLToPDF(r.HTML)
		} else if r.Base64 != "" {
			localPath, err = logic.SaveBase64ToFile(r.Base64)
		} else {
			localPath, err = logic.DownloadFile(r.FileURL)
		}

		if err != nil {
			log.Printf("File preparation failed for job %s: %v", jobId, err)
			logic.UpdateJobStatus(jobId, models.StatusFailed, fmt.Sprintf("Preparation failed: %v", err))
			return
		}
		defer os.Remove(localPath)

		logic.PersistJobFile(jobId, localPath)
		job = logic.GetJobByID(jobId)
		if job == nil || job.Status == models.StatusFailed {
			log.Printf("Job %s failed persistence, aborting print/preview", jobId)
			return
		}

		if r.Preview {
			log.Printf("Previewing file: %s", job.FilePath)
			if err := logic.PreviewFile(job.FilePath); err != nil {
				logic.UpdateJobStatus(jobId, models.StatusFailed, err.Error())
				return
			}
			logic.UpdateJobStatus(jobId, models.StatusCompleted, "")
			return
		}

		logic.UpdateJobStatus(jobId, models.StatusPrinting, "")
		osJobId, err := logic.SilentPrint(localPath, r.Printer)
		if err != nil {
			log.Printf("Background print failed for job %s: %v", jobId, err)
			logic.UpdateJobStatus(jobId, models.StatusFailed, err.Error())
			return
		}

		if osJobId != "" {
			logic.UpdateJobOSID(jobId, osJobId)
			logic.UpdateJobStatus(jobId, models.StatusQueued, "")
		} else {
			logic.UpdateJobStatus(jobId, models.StatusCompleted, "")
		}
	}(req, job.ID)
}

func handleRetryJob(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}
	id := r.URL.Query().Get("id")
	job := logic.GetJobByID(id)
	if job == nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	go func(j *models.PrintJob) {
		logic.UpdateJobStatus(j.ID, models.StatusPrinting, "")
		osJobId, err := logic.SilentPrint(j.FilePath, j.Printer)
		if err != nil {
			logic.UpdateJobStatus(j.ID, models.StatusFailed, err.Error())
			return
		}

		if osJobId != "" {
			logic.UpdateJobOSID(j.ID, osJobId)
			logic.UpdateJobStatus(j.ID, models.StatusQueued, "")
		} else {
			logic.UpdateJobStatus(j.ID, models.StatusCompleted, "")
		}
	}(job)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Retry started"})
}

func handleViewJob(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}
	id := r.URL.Query().Get("id")
	job := logic.GetJobByID(id)
	if job == nil || job.FilePath == "" {
		http.Error(w, "Job or file not found", http.StatusNotFound)
		return
	}
	if err := logic.PreviewFile(job.FilePath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to view: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}
	id := r.URL.Query().Get("id")
	log.Printf("API: [Delete] Request received. ID: %s", id)

	if id == "" {
		// Delete ALL jobs and files
		log.Printf("API: [Delete] No ID provided, clearing ENTIRE queue and files.")
		logic.ClearCompletedJobs()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "All jobs and files deleted"})
		return
	}

	logic.RemoveJob(id)
	log.Printf("API: [Delete] Successfully processed job: %s", id)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleQueue(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}
	json.NewEncoder(w).Encode(logic.GetJobs())
}

func handleClearQueue(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}
	logic.ClearCompletedJobs()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handlePrinters(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}
	printers, err := logic.GetPrinters()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list printers: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(printers)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "running",
		"version": models.AgentVersion,
	})
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL     string `json:"url"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" || req.Version == "" {
		http.Error(w, "url and version required", http.StatusBadRequest)
		return
	}

	result, err := logic.PerformUpdate(req.URL, req.Version)
	if err != nil {
		http.Error(w, fmt.Sprintf("Update failed: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}
