package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

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
	sseManager    *SSEManager    // For SSE events
}

// NewApp creates a new App application struct
func NewApp(mode string) *App {
	app := &App{
		Mode:       mode,
		users:      make(map[string]*User),
		rooms:      make(map[string]*Room),
		chatPool:   NewChatPool(),
		sseManager: NewSSEManager(),
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

		// Notify the invited user via SSE
		inviteData := map[string]interface{}{
			"roomId":   a.currentRoom.ID,
			"roomName": a.currentRoom.Name,
			"inviter":  "Host",
		}
		if err := a.sseManager.SendToClient(userID, EventUserInvited, inviteData); err != nil {
			fmt.Printf("Failed to send invite event to %s: %v\n", userID, err)
		}
	}

	// Note: Host is NOT added to room, host only manages/observes

	return fmt.Sprintf("Successfully invited %s to room %s", user.Name, a.currentRoom.Name)
}

// CreateRoom creates a new room with the given name
func (a *App) CreateRoom(name string) *Room {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.roomCounter++
	roomID := fmt.Sprintf("room_%d", a.roomCounter)
	room := &Room{
		ID:      roomID,
		Name:    name,
		UserIDs: []string{},
	}
	a.rooms[roomID] = room

	// Notify via SSE
	a.sseManager.BroadcastToAll(EventRoomCreated, room)

	fmt.Printf("Room created: %s (%s)\n", name, roomID)
	return room
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

	// Notify via SSE
	a.sseManager.BroadcastToAll(EventUserCreated, user)

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
	user, userExists := a.users[userID]
	room, roomExists := a.rooms[roomID]

	userName := ""
	userInRoom := false
	if userExists {
		userName = user.Name
		if user.RoomID != nil && *user.RoomID == roomID {
			userInRoom = true
		}
	}

	members := make([]string, 0)
	if roomExists {
		members = append(members, room.UserIDs...)
	}

	a.mu.RUnlock()

	if !userExists {
		return "Error: User not found"
	}

	if !roomExists {
		return "Error: Room not found"
	}

	if !userInRoom {
		return "Error: User is not in this room"
	}

	// Add message to chat pool
	chatMsg := a.chatPool.AddMessage(roomID, userID, userName, message)

	// Notify via SSE
	a.sseManager.BroadcastToUsers(members, EventChatMessage, chatMsg, userID)

	fmt.Printf("Chat message from %s in room %s: %s\n", userName, roomID, message)
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

// StartHTTPServer starts the HTTP server for REST API
func (a *App) StartHTTPServer(port string) {
	http.HandleFunc("/api/users", a.handleUsers)
	http.HandleFunc("/api/users/", a.handleUserByID)
	http.HandleFunc("/api/rooms", a.handleRooms)
	http.HandleFunc("/api/invite", a.handleInvite)
	http.HandleFunc("/api/chat", a.handleChat)
	http.HandleFunc("/api/chat/", a.handleChat)
	http.HandleFunc("/api/leave", a.handleLeave)
	http.HandleFunc("/api/sse", a.handleSSE)

	fmt.Printf("Starting HTTP server on port %s\n", port)
	go http.ListenAndServe(":"+port, nil)
}
