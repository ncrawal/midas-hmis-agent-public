# Midas Health HMIS Agent: The Complete Engineering Master Guide

This document is the absolute, definitive "Source of Truth" for the Midas Health HMIS Agent. It is written strictly for developers to understand exactly how the application works under the hood, line-for-line, based exclusively on the actual codebase. It compiles the architecture, dependency list, file-by-file responsibilities, storage logic, and exact engine implementations (HTML->PDF/ZPL and Windows Spooling) into one unified guide.

---

## 1. Application Overview & Architecture

The **Midas Health HMIS Agent** is a specialized background service built for the Health HMIS platform. It acts as an intermediary bridge between a web-based HMIS application and the client's local operating system hardware.

### 1.1 Core Purposes
- **Device Identification**: Harvesting physical hardware MAC addresses securely to authorize, strictly authenticate, and audit workstation access.
- **Silent Headless Printing**: Bypassing standard browser print dialogs to instantly route documents, labels, and receipts directly to physical printers utilizing custom scaling and margins completely invisible to the user.

### 1.2 Technology Stack
- **Framework**: Electron (v30)
- **Backend Server**: Node.js executing Express.js (v4.19) and WebSockets (`ws`).
- **Package Manager**: `pnpm`
- **OS Interop Layers**: Windows Spooler API (`winspool.drv`), PowerShell `Get-Printer`, MacOS/Linux `CUPS` (`lp`, `lpstat`).

---

## 2. Developer Setup & Bootstrapping

We utilize `pnpm` exclusively to manage dependencies efficiently without massive `node_modules` duplication.

### 2.1 Installation & Run Commands
```bash
# 1. Clone & Install dependencies
git clone https://github.com/midas-health/agent.git
cd health-hmis-agent
pnpm install

# 2. Run in Development Mode 
# This leverages 'concurrently' to run both Vite (React UI) and the Electron backend simultaneously.
pnpm run dev

# 3. Clean and Build for Production
# Automatically bundles the React UI and executes electron-builder to output standalone binaries.
pnpm run clean
pnpm run build
```
This produces the final **Midas Health HMIS Agent Setup.exe** (Windows) and **.dmg** (Mac) into the `/release` folder.

---

## 3. Library Information & Dependency Usage

Every dependency in this project has a specific, highly targeted use case:

| Library | Exact Purpose & Case (`package.json`) |
| :--- | :--- |
| **`electron`** (v30) | The core engine. Used for its `BrowserWindow` to spawn "Invisible/Offscreen" Chrome instances that render HTML, execute Javascript calculations in the background, and hook directly into native OS Print Spoolers via `.print()` and `.printToPDF()`. |
| **`express`** (v4.19) | Runs the internal HTTP REST API on `localhost:3033`. It acts as the intake server for all print jobs sent by the React frontend via standard `fetch()` API. |
| **`ws`** | The WebSocket server. Binds to `/ws` and pushes live updates (down to the millisecond) of the `jobs` array back to the React UI so users see exactly when a job enters the "printing" or "completed" state. |
| **`cors`** | Express Middleware. By default, browsers block a website (e.g., `app.midas.com`) from making requests to `localhost`. CORS safely whitelists this local communication. |
| **`body-parser`** | Express Middleware. Configured to explicitly accept massive `50mb` JSON payloads. This is required because the HMIS Web App often passes raw Base64 strings of master PDF reports. |
| **`systeminformation`** | Hardware Interfacing. Used to query the motherboard's Network Interface Cards (NICs), filtering out virtual machines to find the physical Ethernet (`eth0`) or Wifi (`en0`) adapter. It extracts the raw MAC address. |
| **`node-fetch`** | A lightweight network fetcher. Used specifically when the HMIS sends a print job containing a `fileUrl` instead of raw HTML to download the PDF securely. |
| **`uuid`**| Generates a guaranteed-unique `v4` ID for every print job that enters the queue, preventing filename collisions in the OS Temporary Directory. |
| **`electron-builder`** | The compiler toolchain used by `pnpm run build` to package the `.exe` and `.dmg`. |

---

## 4. The Application Lifecycle (`main.js`)

The `main.js` file is the entry point. Its primary job is to ensure the app is stable, runs strictly in the background (via Tray icon), and does not leak memory over a 45+ hour continuous hospital shift.

### 4.1 Key Implementation Details

1.  **Single Instance Lock**: The app is built for hospital terminals. We use `app.requestSingleInstanceLock()` to guarantee that if a user clicks the shortcut 50 times, only ONE background service runs. If a second instance spins up, it immediately kills itself and focuses the original tray window.
2.  **Aggressive Garbage Collection**: Because Electron rendering engines can leak DOM nodes, `main.js` boots with `app.commandLine.appendSwitch('js-flags', '--expose-gc')`. A watchdog runs every 15 minutes:

```javascript
// Memory Monitor Watchdog
setInterval(() => {
    const usage = process.memoryUsage();
    const heapUsedMB = Math.round(usage.heapUsed / 1024 / 1024);
    if (heapUsedMB > 200) { 
        if (global.gc) global.gc(); // Manually clear Chromium's heap
    }
}, 1000 * 60 * 15);
```

3.  **Boot Sequence**: It spins up the tray (`createTray`), registers IPC channels, checks GitHub for updates, and calls `startServer()` on port 3033.

---

## 5. The Local API Bridge (`server.js`) & Developer UI Interaction

This Express server handles requests entirely on `localhost:3033`. It does not accept outside internet traffic.

### 5.1 Local Endpoints
- **`POST /print`**: The brain of the API. Receives `fileUrl`, `base64`, or `html` payload alongside the `printer` name, `type` (`html`, `raw`, `escpos`), and `pageSize`.
- **`GET /health-agent`**: Provides a hardware fingerprint back to the HMIS website (OS platform, Primary MAC Address).
- **`GET /printers`**: Calls into the OS layer to fetch connected physical printers.
- **`GET /queue`, `POST /queue/clear`, `GET /queue/delete`**: Queue management.

### 5.2 Sending a Print Request (React / Frontend Example)
To interact with the agent natively from the Web UI:

```javascript
// Example: Sending HTML to print silently on an A4 Laser Printer
async function printInvoice(htmlContent, printerName) {
  try {
    const response = await fetch('http://localhost:3033/print', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        html: htmlContent,             // Raw HTML string
        printer: printerName,          // Exact name from GET /printers
        type: 'html',                  // Print mode: 'html' | 'raw' | 'escpos'
        orientation: 'portrait',       // 'portrait' | 'landscape'
        pageSize: {
          pageSizeName: 'A4',          // Standard sizes ('A4', 'Legal')
          printBackground: true        
        }
      })
    });
    const result = await response.json();
    console.log('Queued Job ID:', result.jobId);
  } catch (error) {
    console.error('Agent is unreachable:', error);
  }
}
```

---

## 6. Job Storage and File Cleanup Lifecycle (`services/printerManager.js`)

Because we are dealing with high-volume hospital environments, memory and disk management are critical. We do NOT use typical Databases (SQL/Mongo). The entire queue is an in-memory javascript array (`let jobs = []`).

### 6.1 The Watchdog Deletion Loop
Inside `services/printerManager.js`, a `setInterval` runs exactly every 5 minutes:
1.  **Success (`completed`)**: If age is older than 10 Minutes, it executes `fs.unlinkSync(job.filePath)`, destroying the `.pdf` or `.zpl` off the client hard drive.
2.  **Failed / Hung (`failed/pending`)**: If age is older than 24 Hours, it is fully purged and deleted, providing a 24-hour debug window for IT.
3.  **Manual Purge**: If a user clicks **Clear Queue**, it bypasses the timers and instantly deletes all correlated files.

```javascript
// Actual code snippet inside printerManager.js
jobsToRemove.forEach(job => {
    if (job.filePath && fs.existsSync(job.filePath)) {
        fs.unlinkSync(job.filePath); // Cleans up the OS temp folder!
    }
});
```

---

## 7. The Direct Print Pipeline: HTML vs. PDF Conversion

The Agent manages renderers inside `printerManager.js` and `services/pdfService.js`.

### 7.1 Scenario A: Direct HTML Printing (`osPrinter.js`)
- **When it occurs**: The web app sends `html` content, and `preview` is `false`.
- **How it works**:
    - It spawns a hidden `BrowserWindow` (`webPreferences: { offscreen: true }`).
    - The raw HTML is loaded via `data:text/html;charset=utf-8`.
    - JS executes a dynamic CSS override for physical dimensions:
      ```javascript
      @media print {
          @page { size: 100mm 50mm; margin: 0 !important; }
      }
      ```
    - It calls Electron's native `win.webContents.print({ silent: true, deviceName: printerName })`.

### 7.2 Scenario B: HTML to PDF Conversion (`services/pdfService.js`)
- **When it occurs**: The user requests `preview: true`.
- **How it works**:
    - Spawns hidden window and injects `@page` bounds.
    - Instead of calling `.print()`, it calls `.printToPDF()`.
    - Generates a byte array saved to `os.tmpdir()` as `receipt-[id].pdf`.
    - Uses `shell.openPath()` to launch the OS default PDF viewer.

---

## 8. The ZPL Converter Engine (`services/zplService.js`)

HTML cannot be sent natively to thermal sticker printers (blurred graphics). **ZPL (Zebra Programming Language)** is required.

### 8.1 Translation Logic
1.  **DPI & Scale**: Sets physical Target DPI (203 DPI) and a virtual Screen DPI (96 DPI).
2.  **Attribute Scanning**: JS hunts for attributes:
    - `data-barcode="true"` -> ZPL `^BCN` Code 128.
    - `data-qrcode="true"` -> ZPL `^BQN` QR code.
3.  **Math**: Multiplies pixel locations by the DPI scale and a 1.5mm safety margin to calculate physical dot-coordinates.

### 8.2 The Output String Example
```zpl
^XA
^PW600
^LL400
^CI28
^FO10,20^A0N,24,24^FDPatient John Doe^FS
^FO10,80^BCN,60,Y,N,N^FD12345678^FS
^XZ
```

---

## 9. Physical OS Hardware Spooling (`services/osPrinter.js`)

Raw ZPL/ESC-POS assembly cannot be sent through standard print APIs. We heavily intervene here.

### 9.1 Bypassing Drivers on Windows (`sendPhysicalRaw`)
1.  **PowerShell Injection**: Dynamically generates a temporary `.ps1` script.
2.  **C# COM Bridge**: Embeds **C# (`System.Runtime.InteropServices`)**.
3.  **Direct Spooling**: Hooks into **`winspool.drv`**. Calls `OpenPrinter`, allocates memory, and pipes the assembly straight to the USB port.

### 9.2 MacOS/Linux Spooling (`CUPS`)
The agent spawns a child shell using the native raw passthrough of CUPS:
```bash
lp -o raw "/tmp/raw-job-1234.zpl" -d "Zebra_Printer_Name"
```

---
**Document Revision**: 2026.03.18 | Professional Engineering Master Guide for Midas Health.
