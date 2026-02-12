package logic

import (
	"health-hmis-agent/internal/models"
	"net"
	"os"
	"runtime"
)

// GetDeviceInfo uses hardware-level logic to find the primary identity.
func GetDeviceInfo() models.DeviceInfo {
	info := models.DeviceInfo{
		OS:      runtime.GOOS,
		Version: models.AgentVersion,
		MACs:    []string{},
	}

	if host, err := os.Hostname(); err == nil {
		info.Hostname = host
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return info
	}

	contains := func(slice []string, val string) bool {
		for _, item := range slice {
			if item == val {
				return true
			}
		}
		return false
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		mac := iface.HardwareAddr.String()
		if mac == "" {
			continue
		}

		if !contains(info.MACs, mac) {
			info.MACs = append(info.MACs, mac)
		}

		isPriority := iface.Name == "en0" || iface.Name == "eth0"
		if info.MAC == "" || isPriority {
			if info.MAC != "" {
				if isPriority {
					info.MAC = mac
				}
			} else {
				info.MAC = mac
			}
		}

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
