// Helper to safely call Wails functions if they exist
const callWails = async (funcName, ...args) => {
    try {
        const path = funcName.split('.');
        let current = window.go;
        for (const p of path) {
            if (!current) return null;
            current = current[p];
        }
        if (typeof current === 'function') {
            return await current(...args);
        }
    } catch (e) { }
    return null;
};

const hostnameEl = document.getElementById('hostname');
const osEl = document.getElementById('os');
const ipEl = document.getElementById('ip');
const macEl = document.getElementById('mac');
const activePrintersEl = document.getElementById('active-printers');
const storagePathEl = document.getElementById('storage-path');
const versionPortEl = document.getElementById('version-port');
const jobTableBody = document.getElementById('job-table-body');
const clearQueueBtn = document.getElementById('clear-queue');

const previewModal = document.getElementById('preview-modal');
const pdfViewer = document.getElementById('pdf-viewer');

async function updateInfo() {
    try {
        let info = await callWails('ui.App.GetFullInfo');
        if (!info) {
            const resp = await fetch('http://localhost:3033/health-agent');
            info = await resp.json();
        }
        if (!info) return;

        hostnameEl.innerText = info.hostname || 'Unknown';
        osEl.innerText = info.os || 'Unknown';
        ipEl.innerText = info.ip || 'Unknown';
        macEl.innerText = info.mac || 'Unknown';
        versionPortEl.innerText = `v${info.version || '1.2.1'} • Port ${info.port || '3033'}`;

        if (storagePathEl) {
            storagePathEl.innerText = info.storageDir || '~/.health-agent/print_jobs';
        }

        if (info.printers && info.printers.length > 0) {
            activePrintersEl.innerText = info.printers.join(', ');
        } else {
            activePrintersEl.innerText = 'No printers detected';
        }
    } catch (err) {
        console.error("Failed to fetch info:", err);
    }
}

function renderJobs(jobs) {
    if (!jobs || jobs.length === 0) {
        jobTableBody.innerHTML = '<tr><td colspan="5" style="text-align: center; color: var(--text-dim); padding: 2rem;">No print history found</td></tr>';
        return;
    }

    jobTableBody.innerHTML = jobs.map(job => `
        <tr>
            <td style="font-weight: 600;">${job.hospitalNo || 'N/A'}</td>
            <td>${job.userName || 'N/A'}</td>
            <td style="font-size: 0.65rem; color: var(--text-dim);">${job.createdAt}</td>
            <td>
                <span class="status-badge status-${job.status}">${job.status}</span>
                ${job.error ? `<div style="color: #ef4444; font-size: 0.6rem; margin-top: 0.2rem; max-width: 150px; overflow: hidden; text-overflow: ellipsis;">${job.error}</div>` : ''}
            </td>
            <td>
                <div style="display: flex; gap: 0.25rem;">
                    <button class="action-btn" title="View" onclick="viewJob('${job.id}')">👁️</button>
                    <button class="action-btn" title="Retry" onclick="retryJob('${job.id}')">🔄</button>
                    <button class="action-btn" style="color: #ef4444; border-color: rgba(239, 68, 68, 0.2);" title="Delete" onclick="deleteJob('${job.id}')">🗑️</button>
                </div>
            </td>
        </tr>
    `).reverse().join('');
}

// Global Actions
window.retryJob = async (id) => {
    try {
        await fetch(`http://localhost:3033/queue/retry?id=${encodeURIComponent(id)}`);
        alert("Retry signal sent");
    } catch (err) { alert('Retry failed: ' + err); }
};

window.viewJob = async (id) => {
    try {
        const base64 = await callWails('ui.App.GetJobPDF', id);
        if (!base64) throw new Error("Preview only available inside Agent App or data missing");

        pdfViewer.src = `data:application/pdf;base64,${base64}#toolbar=0&navpanes=0&scrollbar=0`;
        previewModal.style.display = 'block';
    } catch (err) {
        alert('Preview failed: ' + err.message);
    }
};

window.closePreview = () => {
    previewModal.style.display = 'none';
    pdfViewer.src = 'about:blank';
};

window.deleteJob = async (id) => {
    console.log("Delete triggered for:", id);
    if (!confirm('Permanently delete this record?')) return;

    try {
        // Since user confirmed 'fromurl' works, we will use the API directly to be safe
        console.log("Attempting delete via API...");
        const url = `http://localhost:3033/queue/delete?id=${encodeURIComponent(id)}`;

        const resp = await fetch(url);
        if (resp.ok) {
            console.log("API Delete success");
            alert("Success: Record deleted!");
            const updated = await fetchJobs();
            renderJobs(updated);
        } else {
            console.error("API Delete failed with status:", resp.status);
            // Fallback to Wails if API failed?
            if (window.go && window.go.ui && window.go.ui.App) {
                await window.go.ui.App.DeleteJob(id);
                alert("Success: Record deleted (via Wails)!");
                renderJobs(await fetchJobs());
            } else {
                throw new Error("HTTP " + resp.status + " while deleting");
            }
        }
    } catch (err) {
        console.error("Delete final error:", err);
        alert('Error deleting: ' + err.message);
    }
};

async function fetchJobs() {
    try {
        const wailsJobs = await callWails('ui.App.GetJobs');
        if (wailsJobs) return wailsJobs;

        const resp = await fetch('http://localhost:3033/queue');
        if (!resp.ok) return [];
        return await resp.json();
    } catch (err) {
        return [];
    }
}

// Events & Init
if (window.runtime && window.runtime.EventsOn) {
    window.runtime.EventsOn('queue_updated', renderJobs);
}

updateInfo();
fetchJobs().then(renderJobs).catch(() => { });

// Clear All
clearQueueBtn.addEventListener('click', async () => {
    if (confirm('FORCE RESET: This will wipe all history and kill stuck active jobs. Continue?')) {
        try {
            if (window.go && window.go.ui && window.go.ui.App) {
                await window.go.ui.App.ClearJobs();
            } else {
                await fetch('http://localhost:3033/queue/clear');
            }
            renderJobs(await fetchJobs());
            alert("Queue cleared successfully");
        } catch (e) { alert("Clear failed: " + e); }
    }
});

// Intervals
setInterval(async () => {
    const jobs = await fetchJobs();
    renderJobs(jobs);
}, 5000);

setInterval(updateInfo, 30000);
