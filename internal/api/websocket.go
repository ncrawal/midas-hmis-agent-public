package api

import (
	"health-hmis-agent/internal/logic"
	"health-hmis-agent/internal/models"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// Clients connected via WebSocket
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex

	// Upgrader to upgrade HTTP connections to WebSocket
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for local agent (or restricted in prod)
		},
	}
)

// handleWebSocket handles WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	// Send initial state immediately
	initialJobs := logic.GetJobs()
	if err := conn.WriteJSON(initialJobs); err != nil {
		log.Printf("Failed to send initial state: %v", err)
		clientsMu.Lock()
		delete(clients, conn)
		clientsMu.Unlock()
		return
	}

	// Keep connection alive/listen (even if we only push)
	for {
		// Read message (blocks) to detect disconnect
		_, _, err := conn.ReadMessage()
		if err != nil {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			break
		}
	}
}

// BroadcastQueueUpdate sends the updated job list to all connected clients
func BroadcastQueueUpdate(jobs []*models.PrintJob) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	// Serialize explicitly if needed, but WriteJSON handles struct
	// We might need to filter sensitive data? No, jobs are fine.

	for conn := range clients {
		if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
			conn.Close()
			delete(clients, conn)
			continue
		}
		if err := conn.WriteJSON(jobs); err != nil {
			log.Printf("Failed to broadcast to client: %v", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}
