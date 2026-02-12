# Midas Health HMIS Agent - Background Service

This is a specialized utility for the Health HMIS platform. It provides two primary functions:
1. **Device Identification**: Allows the web application to identify the host hardware via MAC address for security and auditing.
2. **Silent Printing**: Enables direct, silent printing of PDFs and documents to local printers without browser print dialogs.

## Security & Transparency

To ensure user trust and avoid being flagged by security software, this agent adheres to the following principles:
- **No Background Mining**: Does not perform any cryptocurrency mining or resource-heavy background tasks.
- **No Hidden Installs**: Installation as a service requires explicit user action via the `install` command with administrative privileges.
- **No Auto-Run Without Consent**: The agent only starts automatically if the user has explicitly installed it as a system service.
- **Full Transparency**: All communication is local (localhost:3033) or limited to fetching document URLs provided by your authorized HMIS web portal.
- **No Suspicious Activity**: Does not perform port scanning, DLL injection, or unsolicited outbound connections.

## How to Install on macOS

### 1. Prepare the Binary
Move the `health-hmis-agent` binary to a permanent location. Do not run it from your Downloads folder if you plan to delete it later. A good location is `/usr/local/bin` or a dedicated folder in `/Applications`.

```bash
# Example: Move to /usr/local/bin (recommended)
sudo mv /path/to/downloaded/health-hmis-agent /usr/local/bin/health-hmis-agent
sudo chmod +x /usr/local/bin/health-hmis-agent
```

### 2. Remove Quarantine (Important)
MacOS may block the binary from running as a service if it's quarantined. Run this command:

```bash
sudo xattr -d com.apple.quarantine /usr/local/bin/health-hmis-agent
```
*(If you see "xattr: No such xattr: com.apple.quarantine", that's fine, proceed.)*

### 3. Install the Service
Run the agent with the `install` command. You must use `sudo` to install it as a system service (which allows it to run on boot before login).

```bash
sudo /usr/local/bin/health-hmis-agent install
```
This creates `/Library/LaunchDaemons/HealthHMISAgent.plist`.

### 4. Start the Service
```bash
sudo /usr/local/bin/health-hmis-agent start
```

### 5. Verify it is Running
You can check the status:
```bash
sudo /usr/local/bin/health-hmis-agent status
```
Or check if the process is running:
```bash
ps aux | grep health-hmis-agent
```
```bash
curl http://localhost:3033/health-agent
```

## Local API Endpoints

The agent exposes a local API on port `3033` for your React application:

### 1. Silent Print
**POST** `http://localhost:3033/print`
```json
{
  "fileUrl": "https://example.com/invoice.pdf",
  "printer": "HP_LaserJet_Pro", // Optional: printer name from /printers
  "copies": 1
}
```

### 2. List Printers
**GET** `http://localhost:3033/printers`
Returns an array of strings representing available printer names on the system.

### 3. Agent Status
**GET** `http://localhost:3033/status`
Returns agent health, version, and port info.

### 4. Device Info (Legacy)
**GET** `http://localhost:3033/health-agent`
Returns MAC addresses and hostname for device identification.

## Running with UI (Wails)
You can now run the agent with a GUI for easier status monitoring:
```bash
wails dev
```
or build a production application:
```bash
wails build
```

## How to Test "Auto-Start on Boot"
1. Perform the steps above.
2. Verify the agent is running (`curl http://localhost:3033/health-agent`).
3. Restart your Mac (`sudo reboot`).
4. Wait for the system to boot up. **Do not log in immediately** if you can enable SSH or remote checks, but typically just log in.
5. Open a terminal and run `curl http://localhost:3033/health-agent` or `ps aux | grep health-hmis-agent`. It should be running.

## Uninstalling
To stop and remove the service:

```bash
sudo /usr/local/bin/health-hmis-agent stop
sudo /usr/local/bin/health-hmis-agent uninstall
rm /usr/local/bin/health-hmis-agent
```

## Windows Installation (Always Background & Auto-Boot)

The Windows version is designed to run as a robust background service that stays active even after failure and starts automatically when the computer turns on.

1.  **Open PowerShell as Administrator** (Right-click Start > Terminal/PowerShell (Admin)).
2.  **Install & Start**:
    ```powershell
    .\health-hmis-agent.exe install
    ```
    *This single command installs the agent as a Windows Service, configures it to **Start Automatically on Boot**, and **Starts it immediately**.*

3.  **Verify**:
    Open the "Services" app in Windows and look for **Midas Health HMIS Agent**. Its Status should be `Running` and Startup Type should be `Automatic`.

## Linux Installation
```bash
sudo ./health-hmis-agent install
# Starts automatically and sets up systemd/init auto-boot
```
