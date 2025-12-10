package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSE Event Types
type SSEEventType string

const (
	EventConnected       SSEEventType = "connected"
	EventUserCreated     SSEEventType = "user_created"
	EventUserLeft        SSEEventType = "user_left"
	EventRoomCreated     SSEEventType = "room_created"
	EventRoomDeleted     SSEEventType = "room_deleted"
	EventUserInvited     SSEEventType = "user_invited"
	EventUserJoined      SSEEventType = "user_joined"
	EventChatMessage     SSEEventType = "chat_message"
	EventHeartbeat       SSEEventType = "heartbeat"
	EventClipboardCopied SSEEventType = "clipboard_copied"
	EventClipboardUpdated SSEEventType = "clipboard_updated"
	EventUserOffline     SSEEventType = "user_offline"
)

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Type      SSEEventType `json:"type"`
	Data      interface{}  `json:"data"`
	Timestamp int64        `json:"timestamp"`
}

// SSEClient represents a connected SSE client
type SSEClient struct {
	UserID  string
	Writer  http.ResponseWriter
	Flusher http.Flusher
	mu      sync.Mutex
}

// SSEManager manages SSE connections and broadcasts events
type SSEManager struct {
	clients map[string]*SSEClient
	mu      sync.RWMutex
}

// NewSSEManager creates a new SSE manager
func NewSSEManager() *SSEManager {
	return &SSEManager{
		clients: make(map[string]*SSEClient),
	}
}

// AddClient adds a new SSE client
func (sm *SSEManager) AddClient(userID string, w http.ResponseWriter, flusher http.Flusher) *SSEClient {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	client := &SSEClient{
		UserID:  userID,
		Writer:  w,
		Flusher: flusher,
	}
	sm.clients[userID] = client
	return client
}

// RemoveClient removes an SSE client if it matches the provided client instance
func (sm *SSEManager) RemoveClient(client *SSEClient) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if current, ok := sm.clients[client.UserID]; ok && current == client {
		delete(sm.clients, client.UserID)
	}
}

// SendToClient sends an event to a specific client
func (sm *SSEManager) SendToClient(userID string, eventType SSEEventType, data interface{}) error {
	client, err := sm.getClient(userID)
	if err != nil {
		return err
	}

	if err := client.sendEvent(eventType, data); err != nil {
		sm.RemoveClient(client)
		return fmt.Errorf("failed to send event to client %s: %w", userID, err)
	}

	return nil
}

// BroadcastToAll sends an event to all connected clients
func (sm *SSEManager) BroadcastToAll(eventType SSEEventType, data interface{}) {
	clients := sm.snapshotClients(nil)
	for _, client := range clients {
		if err := client.sendEvent(eventType, data); err != nil {
			fmt.Printf("BroadcastToAll: dropping client %s due to send error: %v\n", client.UserID, err)
			sm.RemoveClient(client)
		}
	}
}

// BroadcastToUsers sends an event to the provided user IDs
func (sm *SSEManager) BroadcastToUsers(userIDs []string, eventType SSEEventType, data interface{}, excludeUserID string) {
	targetSet := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		targetSet[id] = struct{}{}
	}

	filter := func(client *SSEClient) bool {
		if client.UserID == excludeUserID {
			return false
		}
		_, ok := targetSet[client.UserID]
		return ok
	}

	clients := sm.snapshotClients(filter)

	for _, client := range clients {
		if err := client.sendEvent(eventType, data); err != nil {
			fmt.Printf("BroadcastToUsers: dropping client %s due to send error: %v\n", client.UserID, err)
			sm.RemoveClient(client)
		}
	}
}

// IsConnected checks if a user has an active SSE connection
func (sm *SSEManager) IsConnected(userID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, exists := sm.clients[userID]
	return exists
}

// SendHeartbeat sends a heartbeat event if the client is connected
func (sm *SSEManager) SendHeartbeat(userID string) error {
	return sm.SendToClient(userID, EventHeartbeat, map[string]int64{"timestamp": time.Now().Unix()})
}

func (sm *SSEManager) getClient(userID string) (*SSEClient, error) {
	sm.mu.RLock()
	client, exists := sm.clients[userID]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("client %s not connected", userID)
	}

	return client, nil
}

func (sm *SSEManager) snapshotClients(filter func(*SSEClient) bool) []*SSEClient {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	clients := make([]*SSEClient, 0, len(sm.clients))
	for _, client := range sm.clients {
		if filter == nil || filter(client) {
			clients = append(clients, client)
		}
	}

	return clients
}

func (client *SSEClient) sendEvent(eventType SSEEventType, data interface{}) error {
	event := SSEEvent{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	if _, err := fmt.Fprintf(client.Writer, "event: %s\n", eventType); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(client.Writer, "data: %s\n\n", payload); err != nil {
		return err
	}

	client.Flusher.Flush()
	return nil
}

// handleSSE handles Server-Sent Events connections
func (a *App) handleSSE(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId required", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Get flusher for SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Add client to SSE manager
	client := a.sseManager.AddClient(userID, w, flusher)
	fmt.Printf("SSE connected for user: %s\n", userID)

	// Send initial connection event
	if err := a.sseManager.SendToClient(userID, EventConnected, map[string]string{"status": "connected"}); err != nil {
		fmt.Printf("Failed to send initial SSE event for user %s: %v\n", userID, err)
	}

	// Handle connection cleanup
	defer func() {
		a.sseManager.RemoveClient(client)
		fmt.Printf("SSE disconnected for user: %s\n", userID)

		// Check if user has reconnected (has a valid client in SSE manager)
		if a.sseManager.IsConnected(userID) {
			fmt.Printf("User %s reconnected, skipping cleanup\n", userID)
			return
		}

		// Handle user cleanup
		a.mu.Lock()
		user, exists := a.users[userID]
		a.mu.Unlock()

		if exists {
			if user.RoomID != nil {
				a.LeaveRoom(userID)
			}

			a.mu.Lock()
			delete(a.users, userID)
			a.mu.Unlock()

			// Notify others to update their user list
			a.sseManager.BroadcastToAll(EventUserOffline, map[string]string{"userId": userID})
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := a.sseManager.SendHeartbeat(userID); err != nil {
				fmt.Printf("Heartbeat send failed for user %s: %v\n", userID, err)
				return
			}
		}
	}
}
