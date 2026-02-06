package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
)

// DeviceInfo represents the core identity of the hardware.
type DeviceInfo struct {
	MAC      string   `json:"mac"`      // Primary/Best guess MAC
	MACs     []string `json:"mac_list"` // All valid MACs found
	IP       string   `json:"local_ip"`
	Hostname string   `json:"hostname"`
	OS       string   `json:"os_platform"`
	Version  string   `json:"agent_version"`
}

const (
	AgentVersion = "1.2.1-cli"
	DefaultPort  = "51730"
)

// GetDeviceInfo uses hardware-level logic to find the primary identity.
func GetDeviceInfo() DeviceInfo {
	info := DeviceInfo{
		OS:      runtime.GOOS,
		Version: AgentVersion,
		MACs:    []string{},
	}

	if host, err := os.Hostname(); err == nil {
		info.Hostname = host
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return info
	}

	// Helper to check if array contains value
	contains := func(slice []string, val string) bool {
		for _, item := range slice {
			if item == val {
				return true
			}
		}
		return false
	}

	for _, iface := range interfaces {
		// Skip loopback/down (Keep simple logic)
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		mac := iface.HardwareAddr.String()
		if mac == "" {
			continue
		}

		// Add to comprehensive list if unique
		if !contains(info.MACs, mac) {
			info.MACs = append(info.MACs, mac)
		}

		// Priority Logic for Primary MAC ID:
		// 1. Prefer "en0" (macOS Wi-Fi/Ethernet)
		// 2. Prefer "eth0" (Linux Ethernet)
		// 3. Fallback to first found

		isPriority := iface.Name == "en0" || iface.Name == "eth0"
		if info.MAC == "" || isPriority {
			// If we already have a priority MAC, don't overwrite unless this one is ALSO priority?
			// Simpler: If we haven't found a priority one yet, accept this.
			// If this IS a priority one, take it immediately.

			// If we already have a value, only overwrite if current iface is "en0" or "eth0"
			if info.MAC != "" {
				if isPriority {
					info.MAC = mac
				}
			} else {
				// No mac set yet, take this one
				info.MAC = mac
			}
		}

		// Grab IP (Keep similar logic, prioritize first valid non-loopback)
		if info.IP == "" {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						info.IP = ipnet.IP.String()
					}
				}
			}
		}
	}

	return info
}

func main() {
	port := DefaultPort
	if p := os.Getenv("AGENT_PORT"); p != "" {
		port = p
	}

	http.HandleFunc("/health-agent", func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS for all origins (Safe for local agent)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "OPTIONS" {
			return
		}

		info := GetDeviceInfo()
		json.NewEncoder(w).Encode(info)
	})

	fmt.Printf("Midas Agent %s starting on port %s...\n", AgentVersion, port)
	fmt.Printf("Endpoint: http://localhost:%s/health-agent\n", port)

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}
