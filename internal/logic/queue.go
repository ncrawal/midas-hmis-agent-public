package logic

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"health-hmis-agent/internal/models"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	jobs            []*models.PrintJob
	jobsMu          sync.RWMutex
	onUpdate        func([]*models.PrintJob)
	persistencePath string
	jobsDir         string
	lastSyncTime    time.Time
)

func init() {
	home, _ := os.UserHomeDir()
	storageDir := filepath.Join(home, ".health-agent")
	os.MkdirAll(storageDir, 0755)

	persistencePath = filepath.Join(storageDir, "jobs.json")
	jobsDir = filepath.Join(storageDir, "print_jobs")
	os.MkdirAll(jobsDir, 0755)

	LoadJobs()
	cleanupOrphans()
	StartSyncService()
}

func ensureLoadedLocked() {
	info, err := os.Stat(persistencePath)
	if err != nil {
		// File missing might mean it was cleared by another process
		if len(jobs) > 0 {
			jobs = []*models.PrintJob{}
			lastSyncTime = time.Time{}
		}
		return
	}

	if info.ModTime().After(lastSyncTime) {
		log.Printf("Detected disk change in jobs.json, reloading...")
		data, err := os.ReadFile(persistencePath)
		if err == nil {
			var diskJobs []*models.PrintJob
			if err := json.Unmarshal(data, &diskJobs); err == nil {
				jobs = diskJobs
				lastSyncTime = info.ModTime()
			}
		}
	}
}

func SetOnUpdateCallback(cb func([]*models.PrintJob)) {
	onUpdate = cb
}

func LoadJobs() {
	jobsMu.Lock()
	defer jobsMu.Unlock()

	info, err := os.Stat(persistencePath)
	if err != nil {
		return
	}

	data, err := os.ReadFile(persistencePath)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &jobs)
	if err != nil {
		log.Printf("Failed to unmarshal jobs. Reseting...: %v", err)
		jobs = []*models.PrintJob{}
		return
	}
	lastSyncTime = info.ModTime()

	// Sanity check and cleanup on startup
	changed := false
	var validJobs []*models.PrintJob
	for _, job := range jobs {
		// 1. If job was stuck in "printing", it's definitely not printing anymore
		if job.Status == models.StatusPrinting {
			job.Status = models.StatusFailed
			job.Error = "Agent restarted during print"
			changed = true
		}

		// 2. Check if file still exists if it's supposed to
		if job.FilePath != "" {
			if _, err := os.Stat(job.FilePath); os.IsNotExist(err) {
				log.Printf("Job %s file missing: %s", job.ID, job.FilePath)
				job.Status = models.StatusFailed
				job.Error = "Source file missing"
				job.FilePath = ""
				changed = true
			}
		} else if job.Status == models.StatusPending {
			// Pending jobs without a file path are likely corrupted/stuck
			job.Status = models.StatusFailed
			job.Error = "Job data corrupted (no file)"
			changed = true
		}
		validJobs = append(validJobs, job)
	}
	jobs = validJobs

	if changed {
		saveJobsLocked()
		if onUpdate != nil {
			onUpdate(copyJobsLocked())
		}
	}
}

func saveJobsLocked() {
	data, _ := json.Marshal(jobs)
	err := os.WriteFile(persistencePath, data, 0644)
	if err != nil {
		log.Printf("CRITICAL: Failed to save jobs: %v", err)
		return
	}
	if info, err := os.Stat(persistencePath); err == nil {
		lastSyncTime = info.ModTime()
	}
}

func copyJobsLocked() []*models.PrintJob {
	cp := make([]*models.PrintJob, len(jobs))
	for i, j := range jobs {
		val := *j
		cp[i] = &val
	}
	return cp
}

func AddJob(printer, filename, hospitalNo, userName, tempPath string) *models.PrintJob {
	jobsMu.Lock()
	id := uuid.New().String()

	job := &models.PrintJob{
		ID:         id,
		FileName:   filename,
		HospitalNo: hospitalNo,
		UserName:   userName,
		Printer:    printer,
		Status:     models.StatusPending,
		CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
	}

	jobs = append(jobs, job)
	saveJobsLocked()
	current := copyJobsLocked()
	jobsMu.Unlock()

	if tempPath != "" {
		PersistJobFile(id, tempPath)
	}

	if onUpdate != nil {
		onUpdate(current)
	}
	return job
}

func PersistJobFile(id string, tempPath string) {
	persistentPath := filepath.Join(jobsDir, id+".pdf")
	err := copyFile(tempPath, persistentPath)

	jobsMu.Lock()
	found := false
	for _, job := range jobs {
		if job.ID == id {
			if err == nil {
				job.FilePath = persistentPath
			} else {
				log.Printf("Failed to persist job file for %s: %v", id, err)
				job.Status = models.StatusFailed
				job.Error = fmt.Sprintf("Storage failed: %v", err)
			}
			found = true
			break
		}
	}
	if found {
		saveJobsLocked()
	}
	current := copyJobsLocked()
	jobsMu.Unlock()

	if onUpdate != nil {
		onUpdate(current)
	}
}

func UpdateJobStatus(id string, status string, errStr string) {
	var fileToDelete string

	jobsMu.Lock()
	ensureLoadedLocked()
	idx := -1
	for i, job := range jobs {
		if job.ID == id {
			job.Status = status
			job.Error = errStr
			idx = i
			break
		}
	}

	if idx == -1 {
		jobsMu.Unlock()
		return
	}

	// Auto-remove successful jobs
	if status == models.StatusCompleted {
		fileToDelete = jobs[idx].FilePath
		jobs = append(jobs[:idx], jobs[idx+1:]...)
	}

	saveJobsLocked()
	current := copyJobsLocked()
	jobsMu.Unlock()

	// Slow/blocking ops OUTSIDE of lock
	if fileToDelete != "" {
		os.Remove(fileToDelete)
	}

	if onUpdate != nil {
		onUpdate(current)
	}
}

func UpdateJobOSID(id string, osJobID string) {
	jobsMu.Lock()
	ensureLoadedLocked()
	changed := false
	for _, job := range jobs {
		if job.ID == id {
			job.OSJobID = osJobID
			changed = true
			break
		}
	}
	if changed {
		saveJobsLocked()
	}
	current := copyJobsLocked()
	jobsMu.Unlock()

	if onUpdate != nil {
		onUpdate(current)
	}
}

func GetJobs() []*models.PrintJob {
	jobsMu.Lock()
	ensureLoadedLocked()
	changed := false
	for _, job := range jobs {
		if job.FilePath != "" && job.Status != models.StatusFailed {
			if _, err := os.Stat(job.FilePath); os.IsNotExist(err) {
				job.Status = models.StatusFailed
				job.Error = "Source file missing"
				job.FilePath = ""
				changed = true
			}
		}
	}
	if changed {
		saveJobsLocked()
		if onUpdate != nil {
			onUpdate(copyJobsLocked())
		}
	}
	jobsMu.Unlock()

	jobsMu.RLock()
	defer jobsMu.RUnlock()
	return copyJobsLocked()
}

func RemoveJob(id string) {
	var fileToDelete string
	var osJobID string

	jobsMu.Lock()
	ensureLoadedLocked()
	for i, job := range jobs {
		if job.ID == id {
			fileToDelete = job.FilePath
			osJobID = job.OSJobID
			jobs = append(jobs[:i], jobs[i+1:]...)
			break
		}
	}
	saveJobsLocked()
	current := copyJobsLocked()
	jobsMu.Unlock()

	// Perform cancellations and deletions outside lock
	if osJobID != "" {
		CancelOSJob(osJobID)
	}

	if fileToDelete != "" {
		os.Remove(fileToDelete)
	}

	if onUpdate != nil {
		onUpdate(current)
	}
}

func ClearCompletedJobs() {
	var osJobIDs []string

	jobsMu.Lock()
	for _, job := range jobs {
		if job.OSJobID != "" {
			osJobIDs = append(osJobIDs, job.OSJobID)
		}
	}
	jobs = []*models.PrintJob{}
	saveJobsLocked()
	jobsMu.Unlock()

	// 1. Cancel all OS jobs
	for _, osID := range osJobIDs {
		CancelOSJob(osID)
	}

	// 2. Wipe everything in print_jobs directory
	files, _ := filepath.Glob(filepath.Join(jobsDir, "*"))
	for _, f := range files {
		os.Remove(f)
	}

	// 2. Remove the persistence file itself
	os.Remove(persistencePath)

	if onUpdate != nil {
		onUpdate([]*models.PrintJob{})
	}
}

func cleanupOrphans() {
	files, err := filepath.Glob(filepath.Join(jobsDir, "*"))
	if err != nil {
		return
	}

	jobsMu.RLock()
	activeFiles := make(map[string]bool)
	for _, job := range jobs {
		if job.FilePath != "" {
			activeFiles[filepath.Clean(job.FilePath)] = true
		}
	}
	jobsMu.RUnlock()

	for _, f := range files {
		cleanF := filepath.Clean(f)
		if !activeFiles[cleanF] {
			log.Printf("Removing orphan print file: %s", cleanF)
			os.Remove(cleanF)
		}
	}
}

// SyncQueue manually triggers a check of all jobs against the filesystem
func SyncQueue() {
	// Calling GetJobs is sufficient as it now contains the sync logic
	jobs := GetJobs()
	if onUpdate != nil {
		onUpdate(jobs)
	}
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func GetJobByID(id string) *models.PrintJob {
	jobsMu.Lock()
	ensureLoadedLocked()
	defer jobsMu.Unlock()
	for _, job := range jobs {
		if job.ID == id {
			// Return a copy to be thread safe
			cp := *job
			return &cp
		}
	}
	return nil
}

func GetJobPDFBase64(id string) (string, error) {
	// GetJobByID already handles ensureLoadedLocked and locking
	job := GetJobByID(id)
	if job == nil || job.FilePath == "" {
		return "", os.ErrNotExist
	}

	data, err := os.ReadFile(job.FilePath)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func GetJobsDir() string {
	return jobsDir
}

func StartSyncService() {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		for range ticker.C {
			SyncJobStatuses()
		}
	}()
}

func SyncJobStatuses() {
	activeOSJobs, err := CheckOSQueue()
	if err != nil {
		return
	}

	jobsMu.Lock()
	ensureLoadedLocked()
	changed := false

	for _, job := range jobs {
		if job.Status == models.StatusQueued && job.OSJobID != "" {
			if !activeOSJobs[job.OSJobID] {
				log.Printf("OS Job %s finished. Marking internal job %s as completed.", job.OSJobID, job.ID)
				job.Status = models.StatusCompleted
				changed = true
			}
		}
	}

	if changed {
		var filesToDelete []string
		for i := 0; i < len(jobs); {
			if jobs[i].Status == models.StatusCompleted {
				if jobs[i].FilePath != "" {
					filesToDelete = append(filesToDelete, jobs[i].FilePath)
				}
				jobs = append(jobs[:i], jobs[i+1:]...)
			} else {
				i++
			}
		}

		saveJobsLocked()
		current := copyJobsLocked()
		jobsMu.Unlock()

		for _, f := range filesToDelete {
			os.Remove(f)
		}

		if onUpdate != nil {
			onUpdate(current)
		}
	} else {
		jobsMu.Unlock()
	}
}
