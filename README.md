# Health HMIS Agent

# Health HMIS Device Agent

A lightweight, secure native agent designed to uniquely identify devices for the Health HMIS React web app.

## 🚀 The Problem & Solution

**The Problem:** React web apps cannot reliably identify a device. IP addresses change (dynamic IPs) or are masked (VPNs), making them untrustworthy for hardware-level security or session locking.

**The Solution:** This small native agent runs locally on the client machine. It collects hardware-specific data that web browsers cannot access and exposes it through a secure, local-only API.

### 📊 Info Collected
| Field | Purpose |
| :--- | :--- |
| **MAC Address** | Unique hardware identifier (primary key) |
| **Local IP** | Current network context |
| **Hostname** | Device name for user recognition |
| **OS Name** | Operating system identification |

### 🔒 Security
The agent communicates **only** with your React app via `localhost`. It does not send data to any external server. This keeps sensitive hardware information strictly on the user's machine until requested by your authorized application.

---

## ⚙️ How it Works with React

1.  **Installation:** The user downloads and unzips the agent.
2.  **Background Activity:** The agent runs silently and listens on `127.0.0.1:51730`.
3.  **App Verification:** Your React login page makes a request to `http://127.0.0.1:51730/health-agent`.
4.  **Identification:** If the agent responds with a MAC address, the React app can proceed with a hardware-verified session.

---

## 🍎 macOS / 🐧 Linux (Terminal)

### 1. Run the Agent
Right-click the folder and select **"Open in Terminal"**, then:
```bash
./agent-mac-arm64 -server &
```

### 2. Force Restart (Port 51730)
```bash
kill -9 $(lsof -t -i:51730) 2>/dev/null; ./agent-mac-arm64 -server &
```

---

## 🪟 Windows (Command Prompt)

### 1. Run the Agent
```cmd
start agent-windows-amd64.exe -server
```

### 2. Force Stop
```cmd
taskkill /F /IM agent-windows-amd64.exe /T 2>nul
```

---

## 🔍 API Endpoint
**URL:** `http://127.0.0.1:51730/health-agent`

**Sample Response:**
```json
{
  "mac": "00:1a:2b:3c:4d:5e",
  "ip": "192.168.1.50",
  "hostname": "User-MacBook",
  "os": "darwin"
}
```

## 📦 Build Instructions
To build for all platforms:
```bash
./build_release.sh
```