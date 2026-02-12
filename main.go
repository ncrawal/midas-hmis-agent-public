package main

import (
	"embed"
	"health-hmis-agent/internal/api"
	"health-hmis-agent/internal/models"
	"health-hmis-agent/internal/service"
	"health-hmis-agent/internal/ui"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 1. Handle Service commands first (install, uninstall, start, etc.)
	// If it's a service command, it will execute and exit.
	if len(os.Args) > 1 {
		arg := os.Args[1]
		validCommands := map[string]bool{
			"install": true, "uninstall": true, "start": true,
			"stop": true, "status": true, "restart": true,
		}
		if validCommands[arg] {
			if err := service.RunService(); err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	// 2. Start the HTTP server in background (for non-service mode, e.g., desktop app)
	go func() {
		ln, err := net.Listen("tcp", "127.0.0.1:"+models.DefaultPort)
		if err != nil {
			// Already running or port taken
			return
		}
		ln.Close()

		mux := http.NewServeMux()
		api.RegisterHandlers(mux)

		log.Printf("Starting Background API on port %s...", models.DefaultPort)
		if err := http.ListenAndServe("127.0.0.1:"+models.DefaultPort, mux); err != nil {
			log.Printf("API Server failed: %v", err)
		}
	}()

	// 3. Start Wails GUI
	app := ui.NewApp()

	err := wails.Run(&options.App{
		Title:  "Midas Health HMIS Agent",
		Width:  400,
		Height: 500,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
