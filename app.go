package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// NewHistoryPool creates a new history pool
func NewHistoryPool() *HistoryPool {
	return &HistoryPool{
		operations: make(map[string][]*Operation),
		counter:    0,
	}
}

// AddOperation adds an operation to the history pool
func (hp *HistoryPool) AddOperation(roomID string, opType OperationType, itemID string, item *Item) *Operation {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	hp.counter++
	id := fmt.Sprintf("op_%d", hp.counter)

	// Find the last operation ID for parent
	var parentID string
	if ops, exists := hp.operations[roomID]; exists && len(ops) > 0 {
		parentID = ops[len(ops)-1].ID
	}

	op := &Operation{
		ID:        id,
		ParentID:  parentID,
		OpType:    opType,
		ItemID:    itemID,
		Item:      item,
		Timestamp: time.Now().Unix(),
	}

	hp.operations[roomID] = append(hp.operations[roomID], op)
	return op
}

// GetOperations returns all operations for a room since a given operation ID
func (hp *HistoryPool) GetOperations(roomID, sinceID string) []*Operation {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	ops := hp.operations[roomID]
	if ops == nil {
		return []*Operation{}
	}

	if sinceID == "" {
		// Return all
		result := make([]*Operation, len(ops))
		copy(result, ops)
		return result
	}

	// Find index of sinceID
	startIdx := -1
	for i, op := range ops {
		if op.ID == sinceID {
			startIdx = i + 1
			break
		}
	}

	if startIdx == -1 || startIdx >= len(ops) {
		return []*Operation{}
	}

	result := make([]*Operation, len(ops)-startIdx)
	copy(result, ops[startIdx:])
	return result
}

// GetCurrentChatMessages returns current chat messages by applying operations
func (hp *HistoryPool) GetCurrentChatMessages(roomID string) []*ChatMessage {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	ops := hp.operations[roomID]
	messages := make(map[string]*ChatMessage)

	for _, op := range ops {
		if op.Item != nil && op.Item.Type == ItemChat {
			if op.OpType == OpAdd {
				if msg, ok := op.Item.Data.(*ChatMessage); ok {
					messages[op.ItemID] = msg
				}
			} else if op.OpType == OpRemove {
				delete(messages, op.ItemID)
			}
		}
	}

	result := make([]*ChatMessage, 0, len(messages))
	for _, msg := range messages {
		result = append(result, msg)
	}

	// Sort by timestamp
	// Assuming IDs are sequential, but to be safe, sort
	// For now, assume order is preserved
	return result
}

// GetCurrentClipboardItems returns current clipboard items
func (hp *HistoryPool) GetCurrentClipboardItems(roomID string) []*ClipboardItem {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	ops := hp.operations[roomID]
	items := make(map[string]*ClipboardItem)

	for _, op := range ops {
		if op.Item != nil && op.Item.Type == ItemClipboard {
			if op.OpType == OpAdd {
				if item, ok := op.Item.Data.(*ClipboardItem); ok {
					items[op.ItemID] = item
				}
			} else if op.OpType == OpRemove {
				delete(items, op.ItemID)
			}
		}
	}

	result := make([]*ClipboardItem, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
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
	historyPool   *HistoryPool
	mu            sync.RWMutex
	networkClient *NetworkClient // For client mode
	sseManager    *SSEManager    // For SSE events
}

// NewApp creates a new App application struct
func NewApp(mode string) *App {
	app := &App{
		Mode:        mode,
		users:       make(map[string]*User),
		rooms:       make(map[string]*Room),
		historyPool: NewHistoryPool(),
		sseManager:  NewSSEManager(),
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

	// Start clipboard monitoring for copy hotkey
	a.StartClipboardMonitor()
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

// InviteWithRoom invites a user to a room, creating one if needed
// This is used in client mode where users can invite each other
func (a *App) InviteWithRoom(inviteeID, inviterID string) (string, string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if invitee exists
	invitee, exists := a.users[inviteeID]
	if !exists {
		return "", "Error: Invitee not found"
	}

	// Check if inviter exists
	inviter, exists := a.users[inviterID]
	if !exists {
		return "", "Error: Inviter not found"
	}

	// Get or create room for inviter
	var room *Room
	inviterJustJoined := false

	if inviter.RoomID != nil {
		// Inviter already has a room
		room = a.rooms[*inviter.RoomID]
		fmt.Printf("Inviter %s already in room %s\n", inviterID, room.ID)
	} else {
		// Create new room
		a.roomCounter++
		roomID := fmt.Sprintf("room_%d", a.roomCounter)
		room = &Room{
			ID:      roomID,
			Name:    fmt.Sprintf("Room %d", a.roomCounter),
			UserIDs: []string{},
		}
		a.rooms[roomID] = room

		// Add inviter to the room
		room.UserIDs = append(room.UserIDs, inviterID)
		inviter.RoomID = &room.ID
		inviterJustJoined = true

		fmt.Printf("Created new room %s for inviter %s\n", room.ID, inviterID)
	}

	// Always notify inviter to ensure they see the room view
	// This is critical for the first invite when the inviter creates the room
	if inviterJustJoined {
		inviteData := map[string]interface{}{
			"roomId":   room.ID,
			"roomName": room.Name,
			"inviter":  "Self", // This tells the inviter's frontend to auto-join
		}

		fmt.Printf("DEBUG: About to send SSE to inviter %s\n", inviterID)
		fmt.Printf("DEBUG: Invite data: %+v\n", inviteData)

		if err := a.sseManager.SendToClient(inviterID, EventUserInvited, inviteData); err != nil {
			fmt.Printf("ERROR: Failed to send invite event to inviter %s: %v\n", inviterID, err)
		} else {
			fmt.Printf("SUCCESS: Sent SSE 'Self' invite to inviter %s for room %s\n", inviterID, room.ID)
		}
	}

	// Add invitee to room if not already there
	if !contains(room.UserIDs, inviteeID) {
		room.UserIDs = append(room.UserIDs, inviteeID)
		invitee.RoomID = &room.ID

		// Notify the invitee via SSE
		inviteData := map[string]interface{}{
			"roomId":   room.ID,
			"roomName": room.Name,
			"inviter":  inviter.Name,
		}

		fmt.Printf("DEBUG: About to send SSE to invitee %s\n", inviteeID)
		fmt.Printf("DEBUG: Invite data: %+v\n", inviteData)

		if err := a.sseManager.SendToClient(inviteeID, EventUserInvited, inviteData); err != nil {
			fmt.Printf("ERROR: Failed to send invite event to invitee %s: %v\n", inviteeID, err)
		} else {
			fmt.Printf("SUCCESS: Sent SSE invite to invitee %s for room %s\n", inviteeID, room.ID)
		}
	}

	return room.ID, fmt.Sprintf("Successfully invited %s to room %s", invitee.Name, room.Name)
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

// JoinRoom adds a user to a room and notifies all room members
func (a *App) JoinRoom(userID, roomID string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	user, exists := a.users[userID]
	if !exists {
		return "Error: User not found"
	}

	room, exists := a.rooms[roomID]
	if !exists {
		return "Error: Room not found"
	}

	// Check if user is already in the room
	if contains(room.UserIDs, userID) {
		fmt.Printf("User %s already in room %s\n", userID, roomID)
		return fmt.Sprintf("%s is already in room %s", user.Name, room.Name)
	}

	// Add user to room
	room.UserIDs = append(room.UserIDs, userID)
	user.RoomID = &room.ID

	fmt.Printf("User %s joined room %s\n", userID, roomID)

	// Notify all users in the room that someone joined
	joinData := map[string]interface{}{
		"roomId":   room.ID,
		"roomName": room.Name,
		"userId":   userID,
		"userName": user.Name,
	}

	for _, memberID := range room.UserIDs {
		if err := a.sseManager.SendToClient(memberID, EventUserJoined, joinData); err != nil {
			fmt.Printf("ERROR: Failed to send join event to %s: %v\n", memberID, err)
		} else {
			fmt.Printf("SUCCESS: Sent join notification to %s\n", memberID)
		}
	}

	return fmt.Sprintf("%s joined room %s", user.Name, room.Name)
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

	// Create chat message
	msg := &ChatMessage{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()), // unique ID
		RoomID:    roomID,
		UserID:    userID,
		UserName:  userName,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	// Create item
	item := &Item{
		ID:   msg.ID,
		Type: ItemChat,
		Data: msg,
	}

	// Add operation
	op := a.historyPool.AddOperation(roomID, OpAdd, msg.ID, item)

	// Notify via SSE
	a.sseManager.BroadcastToUsers(members, EventChatMessage, op, userID)

	fmt.Printf("Chat message from %s in room %s: %s\n", userName, roomID, message)
	return fmt.Sprintf("Message sent: %s", msg.ID)
}

// GetChatHistory returns chat history for a room
func (a *App) GetChatHistory(roomID string) []*ChatMessage {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Verify room exists
	if _, exists := a.rooms[roomID]; !exists {
		return []*ChatMessage{}
	}

	return a.historyPool.GetCurrentChatMessages(roomID)
}

// GetOperations returns operations for a room since a given ID
func (a *App) GetOperations(roomID, sinceID string) []*Operation {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// For global clipboard, allow roomID == "global"
	if roomID != "global" {
		if _, exists := a.rooms[roomID]; !exists {
			return []*Operation{}
		}
	}

	return a.historyPool.GetOperations(roomID, sinceID)
}

// StartHTTPServer starts the HTTP server for REST API
func (a *App) StartHTTPServer(port string) {
	http.HandleFunc("/api/users", a.handleUsers)
	http.HandleFunc("/api/users/", a.handleUserByID)
	http.HandleFunc("/api/rooms", a.handleRooms)
	http.HandleFunc("/api/invite", a.handleInvite)
	http.HandleFunc("/api/join", a.handleJoinRoom)
	http.HandleFunc("/api/chat", a.handleChat)
	http.HandleFunc("/api/chat/", a.handleChat)
	http.HandleFunc("/api/operations/", a.handleOperations)
	http.HandleFunc("/api/leave", a.handleLeave)
	http.HandleFunc("/api/sse", a.handleSSE)

	fmt.Printf("Starting HTTP server on port %s\n", port)
	go http.ListenAndServe(":"+port, nil)
}
