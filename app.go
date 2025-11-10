package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// User represents a user in the system
type User struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	RoomID   *string `json:"roomId,omitempty"` // nil if not in any room
	IsOnline bool    `json:"isOnline"`
}

// Room represents a collaboration room
type Room struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	UserIDs []string `json:"userIds"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	ID        string `json:"id"`
	RoomID    string `json:"roomId"`
	UserID    string `json:"userId"`
	UserName  string `json:"userName"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// ChatPool manages chat history for all rooms
type ChatPool struct {
	messages map[string][]*ChatMessage // roomID -> messages
	counter  int
	mu       sync.RWMutex
}

// NewChatPool creates a new chat pool
func NewChatPool() *ChatPool {
	return &ChatPool{
		messages: make(map[string][]*ChatMessage),
		counter:  0,
	}
}

// AddMessage adds a message to the chat pool
func (cp *ChatPool) AddMessage(roomID, userID, userName, message string) *ChatMessage {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.counter++
	msg := &ChatMessage{
		ID:        fmt.Sprintf("msg_%d", cp.counter),
		RoomID:    roomID,
		UserID:    userID,
		UserName:  userName,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	cp.messages[roomID] = append(cp.messages[roomID], msg)
	return msg
}

// GetMessages returns all messages for a room
func (cp *ChatPool) GetMessages(roomID string) []*ChatMessage {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	messages := cp.messages[roomID]
	if messages == nil {
		return []*ChatMessage{}
	}

	// Return a copy to prevent external modification
	result := make([]*ChatMessage, len(messages))
	copy(result, messages)
	return result
}

// ClearRoomMessages clears all messages for a room
func (cp *ChatPool) ClearRoomMessages(roomID string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	delete(cp.messages, roomID)
}

// App struct
type App struct {
	ctx           context.Context
	Mode          string
	users         map[string]*User
	rooms         map[string]*Room
	currentUser   *User
	currentRoom   *Room
	roomCounter   int
	chatPool      *ChatPool
	mu            sync.RWMutex
	networkClient *NetworkClient // For client mode
}

// NewApp creates a new App application struct
func NewApp(mode string) *App {
	app := &App{
		Mode:       mode,
		users:      make(map[string]*User),
		rooms:      make(map[string]*Room),
		chatPool:   NewChatPool(),
	}

	// Initialize with a default current user for host mode
	// Note: host user is NOT added to users map, so it won't appear in user lists
	if mode == "host" {
		app.currentUser = &User{
			ID:       "host",
			Name:     "Host Server",
			IsOnline: true,
		}
		// DO NOT add host to users map - host is not a regular user
		// Host manages the server but doesn't participate in rooms
	} else if mode == "client" {
		// Client mode will set currentUser when user logs in
	}

	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Printf("Starting in %s mode\n", a.Mode)

	// Initialize test users only for host mode (but not the host itself)
	if a.Mode == "host" {
		a.CreateUser("Alice")
		a.CreateUser("Bob")
		a.CreateUser("Charlie")
		fmt.Println("Initialized test users: Alice, Bob, Charlie")

		// Start HTTP server for central server functionality
		a.StartHTTPServer("8080")
	} else if a.Mode == "client" {
		// Initialize network client for client mode
		a.networkClient = NewNetworkClient("http://localhost:8080")

		// Try to connect to server
		if err := a.networkClient.ConnectToServer(); err != nil {
			fmt.Printf("Warning: Could not connect to server: %v\n", err)
			fmt.Println("Please make sure the host is running in host mode")
		} else {
			// Start automatic sync every 10 seconds (reduced from 5)
			a.networkClient.StartAutoSync(10 * time.Second)
			fmt.Println("Connected to central server and started auto-sync")
		}
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// ListAllUsers returns a list of all users in the system
func (a *App) ListAllUsers() []*User {
	a.mu.RLock()
	defer a.mu.RUnlock()

	users := make([]*User, 0, len(a.users))
	for _, user := range a.users {
		users = append(users, user)
	}
	return users
}

// Invite invites a user to the current room
// If no current room exists, creates a new one
func (a *App) Invite(userID string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if we're in host mode
	if a.Mode != "host" {
		return "Error: Invite can only be used in host mode"
	}

	// Check if user exists
	user, exists := a.users[userID]
	if !exists {
		return "Error: User not found"
	}

	// If no current room, create a new one
	if a.currentRoom == nil {
		a.roomCounter++
		roomID := fmt.Sprintf("room_%d", a.roomCounter)
		a.currentRoom = &Room{
			ID:      roomID,
			Name:    fmt.Sprintf("Room %d", a.roomCounter),
			UserIDs: []string{},
		}
		a.rooms[roomID] = a.currentRoom
	}

	// Add user to current room if not already there
	if !contains(a.currentRoom.UserIDs, userID) {
		a.currentRoom.UserIDs = append(a.currentRoom.UserIDs, userID)
		user.RoomID = &a.currentRoom.ID
	}

	// Note: Host is NOT added to room, host only manages/observes

	return fmt.Sprintf("Successfully invited %s to room %s", user.Name, a.currentRoom.Name)
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// CreateUser creates a new user in the system
func (a *App) CreateUser(name string) *User {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Generate user ID based on current user count
	userID := fmt.Sprintf("user_%d", len(a.users)+1)
	user := &User{
		ID:       userID,
		Name:     name,
		IsOnline: true,
	}
	a.users[userID] = user
	fmt.Printf("Created user: %s (ID: %s)\n", name, userID)
	return user
}

// GetCurrentRoom returns information about the current room
func (a *App) GetCurrentRoom() *Room {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentRoom
}

// GetAllRooms returns all rooms (for debugging/admin purposes)
func (a *App) GetAllRooms() []*Room {
	a.mu.RLock()
	defer a.mu.RUnlock()

	rooms := make([]*Room, 0, len(a.rooms))
	for _, room := range a.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

// GetMode returns the current application mode
func (a *App) GetMode() string {
	return a.Mode
}

// GetConnectionStatus returns whether the client is connected to the server
func (a *App) GetConnectionStatus() bool {
	if a.networkClient == nil {
		return false
	}
	return a.networkClient.IsConnected()
}

// SyncFromServer manually triggers a data sync from the server (client mode only)
func (a *App) SyncFromServer() error {
	if a.Mode != "client" || a.networkClient == nil {
		return fmt.Errorf("sync only available in client mode")
	}
	return a.networkClient.SyncData()
}

// GetServerUsers fetches users from the server (client mode)
func (a *App) GetServerUsers() ([]*User, error) {
	if a.Mode != "client" || a.networkClient == nil {
		return nil, fmt.Errorf("this function is only available in client mode")
	}
	return a.networkClient.FetchUsers()
}

// LeaveRoom removes a user from their current room
// If room has less than 2 people after leaving, the room is deleted
func (a *App) LeaveRoom(userID string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	user, exists := a.users[userID]
	if !exists {
		return "Error: User not found"
	}

	if user.RoomID == nil {
		return "Error: User is not in any room"
	}

	room, roomExists := a.rooms[*user.RoomID]
	if !roomExists {
		return "Error: Room not found"
	}

	// Remove user from room
	for i, uid := range room.UserIDs {
		if uid == userID {
			room.UserIDs = append(room.UserIDs[:i], room.UserIDs[i+1:]...)
			break
		}
	}

	// Clear user's room reference
	user.RoomID = nil

	// If room has less than 2 users, delete it
	if len(room.UserIDs) < 2 {
		delete(a.rooms, room.ID)
		// Remove room reference from remaining users
		for _, uid := range room.UserIDs {
			if u, exists := a.users[uid]; exists {
				u.RoomID = nil
			}
		}
		// If this was the current room, clear it
		if a.currentRoom != nil && a.currentRoom.ID == room.ID {
			a.currentRoom = nil
		}
		return fmt.Sprintf("Room %s deleted: insufficient users after %s left", room.Name, user.Name)
	}

	return fmt.Sprintf("%s left room %s", user.Name, room.Name)
}

// SendChatMessage sends a chat message to a room
func (a *App) SendChatMessage(roomID, userID, message string) string {
	a.mu.RLock()
	user, exists := a.users[userID]
	a.mu.RUnlock()

	if !exists {
		return "Error: User not found"
	}

	if user.RoomID == nil || *user.RoomID != roomID {
		return "Error: User is not in this room"
	}

	a.mu.RLock()
	_, roomExists := a.rooms[roomID]
	a.mu.RUnlock()

	if !roomExists {
		return "Error: Room not found"
	}

	// Add message to chat pool
	chatMsg := a.chatPool.AddMessage(roomID, userID, user.Name, message)

	fmt.Printf("Chat message from %s in room %s: %s\n", user.Name, roomID, message)
	return fmt.Sprintf("Message sent: %s", chatMsg.ID)
}

// GetChatHistory returns chat history for a room
func (a *App) GetChatHistory(roomID string) []*ChatMessage {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Verify room exists
	if _, exists := a.rooms[roomID]; !exists {
		return []*ChatMessage{}
	}

	return a.chatPool.GetMessages(roomID)
}

// handleUsersAndCreate handles GET /api/users (list all users) and POST /api/users (create user)
func (a *App) handleUsersAndCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method == "GET" {
		users := a.ListAllUsers()
		json.NewEncoder(w).Encode(users)
		return
	}

	if r.Method == "POST" {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Check if username is unique
		a.mu.RLock()
		for _, user := range a.users {
			if user.Name == req.Name {
				a.mu.RUnlock()
				http.Error(w, "Username already exists", http.StatusConflict)
				return
			}
		}
		a.mu.RUnlock()

		user := a.CreateUser(req.Name)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleUser handles POST /api/users (create user) and GET /api/users/{id}
func (a *App) handleUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	// Extract user ID from URL path
	path := r.URL.Path
	userID := ""
	if len(path) > len("/api/users/") {
		userID = path[len("/api/users/"):]
	}

	if r.Method == "GET" && userID != "" {
		a.mu.RLock()
		user, exists := a.users[userID]
		a.mu.RUnlock()

		if !exists {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(user)
		return
	}

	if r.Method == "POST" && userID == "" {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Check if username is unique
		a.mu.RLock()
		for _, user := range a.users {
			if user.Name == req.Name {
				a.mu.RUnlock()
				http.Error(w, "Username already exists", http.StatusConflict)
				return
			}
		}
		a.mu.RUnlock()

		user := a.CreateUser(req.Name)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleRooms handles GET /api/rooms
func (a *App) handleRooms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rooms := a.GetAllRooms()
	json.NewEncoder(w).Encode(rooms)
}

// handleInvite handles POST /api/invite
func (a *App) handleInvite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result := a.Invite(req.UserID)
	response := map[string]string{"message": result}
	json.NewEncoder(w).Encode(response)
}

// handleChat handles POST /api/chat (send message) and GET /api/chat/{roomId}
func (a *App) handleChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	// Extract room ID from URL path
	path := r.URL.Path
	roomID := ""
	if len(path) > len("/api/chat/") {
		roomID = path[len("/api/chat/"):]
	}

	if r.Method == "GET" && roomID != "" {
		messages := a.GetChatHistory(roomID)
		json.NewEncoder(w).Encode(messages)
		return
	}

	if r.Method == "POST" {
		var req struct {
			RoomID  string `json:"roomId"`
			UserID  string `json:"userId"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		result := a.SendChatMessage(req.RoomID, req.UserID, req.Message)
		response := map[string]string{"message": result}
		json.NewEncoder(w).Encode(response)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// StartHTTPServer starts the HTTP server for REST API
func (a *App) StartHTTPServer(port string) {
	http.HandleFunc("/api/users", a.handleUsersAndCreate)
	http.HandleFunc("/api/users/", a.handleUser)
	http.HandleFunc("/api/rooms", a.handleRooms)
	http.HandleFunc("/api/invite", a.handleInvite)
	http.HandleFunc("/api/chat", a.handleChat)
	http.HandleFunc("/api/chat/", a.handleChat)

	fmt.Printf("Starting HTTP server on port %s\n", port)
	go http.ListenAndServe(":"+port, nil)
}
