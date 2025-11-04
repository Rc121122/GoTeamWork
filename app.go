package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// User represents a user in the system
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RoomID   *string `json:"roomId,omitempty"` // nil if not in any room
	IsOnline bool   `json:"isOnline"`
}

// Room represents a collaboration room
type Room struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	UserIDs []string `json:"userIds"`
}

// App struct
type App struct {
	ctx         context.Context
	Mode        string
	users       map[string]*User
	rooms       map[string]*Room
	currentUser *User
	currentRoom *Room
	roomCounter int
	mu          sync.RWMutex
}

// NewApp creates a new App application struct
func NewApp(mode string) *App {
	app := &App{
		Mode:  mode,
		users: make(map[string]*User),
		rooms: make(map[string]*Room),
	}

	// Initialize with a default current user for host mode
	if mode == "host" {
		app.currentUser = &User{
			ID:       "host",
			Name:     "Host User",
			IsOnline: true,
		}
		app.users["host"] = app.currentUser
	}

	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Printf("Starting in %s mode\n", a.Mode)

	// Initialize test users in host mode
	if a.Mode == "host" {
		a.CreateUser("Alice")
		a.CreateUser("Bob")
		a.CreateUser("Charlie")
		fmt.Println("Initialized test users: Alice, Bob, Charlie")

		// Start HTTP server for central server functionality
		a.StartHTTPServer("8080")
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

	// Add current user to room if not already there
	if !contains(a.currentRoom.UserIDs, a.currentUser.ID) {
		a.currentRoom.UserIDs = append(a.currentRoom.UserIDs, a.currentUser.ID)
		a.currentUser.RoomID = &a.currentRoom.ID
	}

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

	userID := fmt.Sprintf("user_%d", len(a.users)+1)
	user := &User{
		ID:       userID,
		Name:     name,
		IsOnline: true,
	}
	a.users[userID] = user
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

// StartHTTPServer starts the HTTP server for REST API
func (a *App) StartHTTPServer(port string) {
	http.HandleFunc("/api/users", a.handleUsersAndCreate)
	http.HandleFunc("/api/users/", a.handleUser)
	http.HandleFunc("/api/rooms", a.handleRooms)
	http.HandleFunc("/api/invite", a.handleInvite)

	fmt.Printf("Starting HTTP server on port %s\n", port)
	go http.ListenAndServe(":"+port, nil)
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
