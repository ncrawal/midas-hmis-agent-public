package service

import (
	"fmt"
	"health-hmis-agent/internal/api"
	"health-hmis-agent/internal/models"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/kardianos/service"
)

type program struct {
	exit chan struct{}
}

func (p *program) Start(s service.Service) error {
	p.exit = make(chan struct{})
	go p.run()
	return nil
}

func (p *program) run() {
	ln, err := net.Listen("tcp", "127.0.0.1:"+models.DefaultPort)
	if err != nil {
		log.Printf("Agent is already running: %v", err)
		return
	}
	ln.Close()

	// Start the HTTP server
	mux := http.NewServeMux()
	api.RegisterHandlers(mux)

	log.Printf("Health HMIS Agent Server starting on port %s...", models.DefaultPort)
	if err := http.ListenAndServe("127.0.0.1:"+models.DefaultPort, mux); err != nil {
		log.Printf("Server failed: %v", err)
	}

	<-p.exit
}

func (p *program) Stop(s service.Service) error {
	close(p.exit)
	return nil
}

func GetConfig() *service.Config {
	return &service.Config{
		Name:        "HealthHMISAgent",
		DisplayName: "Midas Health HMIS Agent",
		Description: "Midas Health HMIS Device Identification & Silent Print Agent",
		Option: service.KeyValue{
			"RunAtLoad":        true,        // macOS
			"DelayedAutoStart": false,       // Windows: start as soon as possible
			"StartType":        "automatic", // Windows: ensure it starts on boot
		},
	}
}

func RunService() error {
	prg := &program{}
	s, err := service.New(prg, GetConfig())
	if err != nil {
		return err
	}

	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "install" || arg == "uninstall" || arg == "start" || arg == "stop" || arg == "status" || arg == "restart" {
			if arg == "status" {
				status, err := s.Status()
				if err != nil {
					fmt.Printf("Error getting status: %v\n", err)
				}
				fmt.Printf("Service status: %v\n", status)
				return nil
			}
			err = service.Control(s, arg)
			if err != nil {
				return fmt.Errorf("service command failed: %v", err)
			}
			fmt.Printf("Service %sed successfully\n", arg)

			// Proactive: If we just installed, try to start it immediately too
			if arg == "install" {
				fmt.Println("🚀 Starting service immediately...")
				service.Control(s, "start")
			}
			return nil
		}
	}

	return s.Run()
}
