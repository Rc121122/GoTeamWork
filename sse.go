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
	EventUserCreated  SSEEventType = "user_created"
	EventUserLeft     SSEEventType = "user_left"
	EventRoomCreated  SSEEventType = "room_created"
	EventRoomDeleted  SSEEventType = "room_deleted"
	EventUserInvited  SSEEventType = "user_invited"
	EventChatMessage  SSEEventType = "chat_message"
	EventHeartbeat    SSEEventType = "heartbeat"
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
func (sm *SSEManager) AddClient(userID string, w http.ResponseWriter, flusher http.Flusher) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.clients[userID] = &SSEClient{
		UserID:  userID,
		Writer:  w,
		Flusher: flusher,
	}
}

// RemoveClient removes an SSE client
func (sm *SSEManager) RemoveClient(userID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.clients, userID)
}

// SendToClient sends an event to a specific client
func (sm *SSEManager) SendToClient(userID string, eventType SSEEventType, data interface{}) error {
	sm.mu.RLock()
	client, exists := sm.clients[userID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client not connected")
	}

	event := SSEEvent{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Send SSE event
	fmt.Fprintf(client.Writer, "event: %s\n", eventType)
	fmt.Fprintf(client.Writer, "data: %s\n\n", jsonData)
	client.Flusher.Flush()

	return nil
}

// BroadcastToAll sends an event to all connected clients
func (sm *SSEManager) BroadcastToAll(eventType SSEEventType, data interface{}) {
	sm.mu.RLock()
	clients := make(map[string]*SSEClient)
	for k, v := range sm.clients {
		clients[k] = v
	}
	sm.mu.RUnlock()

	event := SSEEvent{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	for _, client := range clients {
		jsonData, _ := json.Marshal(event)
		fmt.Fprintf(client.Writer, "event: %s\n", eventType)
		fmt.Fprintf(client.Writer, "data: %s\n\n", jsonData)
		client.Flusher.Flush()
	}
}

// BroadcastToRoom sends an event to all clients in a specific room
func (sm *SSEManager) BroadcastToRoom(roomID string, eventType SSEEventType, data interface{}, excludeUserID string) {
	sm.mu.RLock()
	clients := make(map[string]*SSEClient)
	for k, v := range sm.clients {
		clients[k] = v
	}
	sm.mu.RUnlock()

	for userID, client := range clients {
		if userID == excludeUserID {
			continue
		}
		// Note: In a real implementation, you'd check if user is in the room
		// For now, broadcast to all except sender
		event := SSEEvent{
			Type:      eventType,
			Data:      data,
			Timestamp: time.Now().Unix(),
		}

		jsonData, _ := json.Marshal(event)
		fmt.Fprintf(client.Writer, "event: %s\n", eventType)
		fmt.Fprintf(client.Writer, "data: %s\n\n", jsonData)
		client.Flusher.Flush()
	}
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
	a.sseManager.AddClient(userID, w, flusher)
	fmt.Printf("SSE connected for user: %s\n", userID)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	// Handle connection cleanup
	defer func() {
		a.sseManager.RemoveClient(userID)
		fmt.Printf("SSE disconnected for user: %s\n", userID)
	}()

	// Keep connection alive
	for {
		select {
		case <-r.Context().Done():
			return
		default:
			// Send heartbeat every 30 seconds to keep connection alive
			time.Sleep(30 * time.Second)
			fmt.Fprintf(w, "event: heartbeat\ndata: {\"timestamp\":%d}\n\n", time.Now().Unix())
			flusher.Flush()
		}
	}
}