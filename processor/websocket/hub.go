package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Dashboard is same-origin in production; kept permissive for local dev.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub tracks connected dashboard clients and fans a message out to all of
// them. One process can hold thousands of idle websocket connections, so
// this stays simple (no per-client goroutine pools needed at this scale).
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
	// broadcast carries pre-serialized JSON payloads for every tick/event.
	broadcast chan []byte
	register  chan *websocket.Conn
	unregister chan *websocket.Conn
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				_ = conn.Close()
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					// Let the read loop's defer handle unregistering; just skip.
					continue
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast queues a JSON payload to be sent to every connected dashboard.
func (h *Hub) Broadcast(payload []byte) {
	select {
	case h.broadcast <- payload:
	default:
		log.Println("broadcast channel full, dropping tick")
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	h.register <- conn

	// Reader goroutine: we don't expect messages from the dashboard, but we
	// need to keep reading to detect disconnects and respond to pings.
	go func() {
		defer func() { h.unregister <- conn }()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}
