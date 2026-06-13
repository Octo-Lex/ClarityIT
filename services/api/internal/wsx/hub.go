package wsx

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/clarityit/api/internal/iam"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub manages WebSocket connections scoped to teams.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{} // teamID → clients
}

// Client represents an authenticated WebSocket connection.
type Client struct {
	Hub    *Hub
	TeamID string
	UserID string
	Conn   *websocket.Conn
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]struct{}),
	}
}

// HandleWS upgrades an authenticated HTTP connection to WebSocket.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	// Auth check — claims set by middleware
	claims, ok := r.Context().Value("claims").(*iam.TokenClaims)
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.TeamID == "" {
		http.Error(w, "No team context", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade: %v", err)
		return
	}

	client := &Client{
		Hub:    h,
		TeamID: claims.TeamID,
		UserID: claims.UserID,
		Conn:   conn,
	}

	h.Register(client)

	// Read loop (discard, keep alive)
	go func() {
		defer h.Unregister(client)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// Register adds a client to the hub.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.TeamID] == nil {
		h.clients[c.TeamID] = make(map[*Client]struct{})
	}
	h.clients[c.TeamID][c] = struct{}{}
	log.Printf("WS client registered: team=%s user=%s", c.TeamID, c.UserID)
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[c.TeamID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.clients, c.TeamID)
		}
	}
	c.Conn.Close()
	log.Printf("WS client unregistered: team=%s user=%s", c.TeamID, c.UserID)
}

// BroadcastToTeam sends a JSON message to all clients in a team.
func (h *Hub) BroadcastToTeam(teamID string, msg any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.clients[teamID]
	if !ok {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for c := range clients {
		if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			// Connection is stale; let read loop clean it up
		}
	}
}

// SubscribeRedis fans out messages from a channel to WebSocket clients.
func (h *Hub) SubscribeRedis(ch <-chan string) {
	go func() {
		for payload := range ch {
			var parsed struct {
				TeamID string          `json:"team_id"`
				Event  json.RawMessage `json:"event"`
			}
			if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
				continue
			}
			if parsed.TeamID != "" {
				h.BroadcastToTeam(parsed.TeamID, map[string]any{
					"type": "event",
					"data": json.RawMessage(parsed.Event),
				})
			}
		}
	}()
}

// Redis fanout helpers — wire go-redis subscriptions into SubscribeRedis channel.
