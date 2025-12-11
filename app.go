package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"GOproject/clip_helper"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	maxOperationsPerRoom   = 1000             // Maximum operations to keep per room
	maxChatMessagesPerRoom = 100              // Maximum chat messages to keep per room
	roomCleanupInterval    = 30 * time.Minute // Check for empty rooms every 30 minutes
	userTimeout            = 24 * time.Hour   // Remove inactive users after 24 hours
	inviteTimeout          = 30 * time.Second // Pending invites expire after 30 seconds
)

// NewHistoryPool creates a new history pool
func NewHistoryPool() *HistoryPool {
	return &HistoryPool{
		operations: make(map[string][]*Operation),
		counter:    0,
	}
}

// AddOperation adds an operation to the history pool with size limits
func (hp *HistoryPool) AddOperation(roomID string, opType OperationType, itemID string, item *Item, userID, userName string) *Operation {
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
		UserID:    userID,
		UserName:  userName,
	}

	// Add operation to room history
	hp.operations[roomID] = append(hp.operations[roomID], op)

	// Enforce size limits
	hp.enforceLimits(roomID)

	return op
}

// enforceLimits ensures room operation counts stay within limits
func (hp *HistoryPool) enforceLimits(roomID string) {
	ops := hp.operations[roomID]
	if len(ops) <= maxOperationsPerRoom {
		return
	}

	// Keep only the most recent operations
	keepStart := len(ops) - maxOperationsPerRoom
	hp.operations[roomID] = ops[keepStart:]

	fmt.Printf("Trimmed operations for room %s to %d (removed %d old operations)\n",
		roomID, maxOperationsPerRoom, keepStart)
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

// GetCurrentChatMessages returns current chat messages by applying operations (limited to recent messages)
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

	// Sort by timestamp (simple bubble sort for now)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Timestamp > result[j].Timestamp {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// Limit to most recent messages
	if len(result) > maxChatMessagesPerRoom {
		start := len(result) - maxChatMessagesPerRoom
		result = result[start:]
		fmt.Printf("Limited chat history for room %s to %d messages\n", roomID, maxChatMessagesPerRoom)
	}

	return result
}

// GetCurrentClipboardItems returns current clipboard items
func (hp *HistoryPool) GetCurrentClipboardItems(roomID string) []*clip_helper.ClipboardItem {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	ops := hp.operations[roomID]
	items := make(map[string]*clip_helper.ClipboardItem)

	for _, op := range ops {
		if op.Item != nil && op.Item.Type == ItemClipboard {
			if op.OpType == OpAdd {
				if item, ok := op.Item.Data.(*clip_helper.ClipboardItem); ok {
					items[op.ItemID] = item
				}
			} else if op.OpType == OpRemove {
				delete(items, op.ItemID)
			}
		}
	}

	result := make([]*clip_helper.ClipboardItem, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

// App struct
type App struct {
	ctx            context.Context
	Mode           string
	users          map[string]*User
	rooms          map[string]*Room
	currentUser    *User
	currentRoom    *Room
	userCounter    int
	roomCounter    int
	inviteCounter  int
	historyPool    *HistoryPool
	mu             sync.RWMutex
	networkClient  *NetworkClient // For client mode
	sseManager     *SSEManager    // For SSE events
	pendingInvites map[string]*PendingInvite

	clipboardMonitorOnce  sync.Once
	clipboardHotkeyCancel context.CancelFunc
	pendingClipboardMu    sync.Mutex
	pendingClipboardItem  *clip_helper.ClipboardItem
	pendingClipboardAt    time.Time
}

const (
	wailsEventClipboardShowButton = "clipboard:show-share-button"
	wailsEventClipboardPermission = "clipboard:permission-state"
)

// NewApp creates a new App application struct
func NewApp(mode string) *App {
	app := &App{
		Mode:           mode,
		users:          make(map[string]*User),
		rooms:          make(map[string]*Room),
		pendingInvites: make(map[string]*PendingInvite),
		historyPool:    NewHistoryPool(),
		sseManager:     NewSSEManager(),
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

		// Start cleanup goroutines for host mode
		go a.startCleanupTasks(ctx)
	} else if a.Mode == "client" {
		// Initialize network client for client mode
		a.networkClient = NewNetworkClient("http://localhost:8080")

		// Try to connect to server
		if err := a.networkClient.ConnectToServer(); err != nil {
			fmt.Printf("Warning: Could not connect to server: %v\n", err)
			fmt.Println("Please make sure the host is running in host mode")
		} else {
			fmt.Println("Connected to central server; streaming updates over SSE")
		}
	}

	// Start clipboard monitoring for copy hotkey
	a.StartClipboardMonitor()
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// SetUser sets the current user with a specific ID and name (used for client mode sync)
func (a *App) SetUser(id, name string) *User {
	a.mu.Lock()
	defer a.mu.Unlock()

	user := &User{
		ID:       id,
		Name:     name,
		IsOnline: true,
	}
	a.users[id] = user
	a.currentUser = user
	return user
}

// ListAllUsers returns a list of all users in the system
func (a *App) ListAllUsers() []*User {
	a.mu.RLock()
	defer a.mu.RUnlock()

	users := make([]*User, 0, len(a.users))
	for _, user := range a.users {
		users = append(users, user)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})

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
			OwnerID: "host", // Host is the owner
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

	cleanName := sanitizeRoomName(name)
	if cleanName == "" {
		cleanName = fmt.Sprintf("Room %d", a.roomCounter+1)
	}

	a.roomCounter++
	roomID := fmt.Sprintf("room_%d", a.roomCounter)
	room := &Room{
		ID:      roomID,
		Name:    cleanName,
		OwnerID: "host", // Default to host for now, or we need to pass creator ID
		UserIDs: []string{},
	}
	a.rooms[roomID] = room

	// Notify via SSE
	a.sseManager.BroadcastToAll(EventRoomCreated, room)

	fmt.Printf("Room created: %s (%s)\n", name, roomID)
	return room
}

// InviteWithRoom now creates a pending invite that is fulfilled once the invitee accepts.
// It no longer creates rooms eagerly; rooms are created when the invite is accepted.
func (a *App) InviteWithRoom(inviteeID, inviterID, message string) (string, string, int64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	invitee, ok := a.users[inviteeID]
	if !ok {
		return "", "Error: Invitee not found", 0
	}

	inviter, ok := a.users[inviterID]
	if !ok {
		return "", "Error: Inviter not found", 0
	}

	if inviter.RoomID != nil {
		return "", "Error: Inviter already in a room", 0
	}

	if invitee.RoomID != nil {
		return "", "Error: Invitee already in a room", 0
	}

	cleanMessage := sanitizeInviteMessage(message)
	if cleanMessage == "" {
		cleanMessage = fmt.Sprintf("Hi, it's me, %s.", inviter.Name)
	}

	a.inviteCounter++
	inviteID := fmt.Sprintf("invite_%d", a.inviteCounter)
	createdAt := time.Now()
	expiresAt := createdAt.Add(inviteTimeout)
	pending := &PendingInvite{
		ID:        inviteID,
		InviterID: inviterID,
		InviteeID: inviteeID,
		Message:   cleanMessage,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}
	a.pendingInvites[inviteID] = pending

	payload := map[string]interface{}{
		"inviteId":  inviteID,
		"inviterId": inviter.ID,
		"inviter":   inviter.Name,
		"message":   cleanMessage,
		"expiresAt": expiresAt.Unix(),
	}
	fmt.Printf("Sending invite %s from %s (%s) to %s (%s)\n", inviteID, inviter.Name, inviterID, invitee.Name, inviteeID)
	if err := a.sseManager.SendToClient(inviteeID, EventUserInvited, payload); err != nil {
		delete(a.pendingInvites, inviteID)
		fmt.Printf("ERROR: Failed to deliver invite to %s: %v\n", inviteeID, err)
		return "", fmt.Sprintf("Invite queued for %s but SSE delivery failed", invitee.Name), 0
	}

	return inviteID, fmt.Sprintf("Invitation sent to %s", invitee.Name), expiresAt.Unix()
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

// AcceptInvite converts a pending invite into an active room when the invitee accepts.
func (a *App) AcceptInvite(inviteID, inviteeID string) (string, string) {
	a.mu.Lock()
	pending, exists := a.pendingInvites[inviteID]
	if !exists {
		a.mu.Unlock()
		return "", "Error: Invite not found"
	}

	if pending.InviteeID != inviteeID {
		a.mu.Unlock()
		return "", "Error: Invite does not belong to this user"
	}

	if time.Now().After(pending.ExpiresAt) {
		delete(a.pendingInvites, inviteID)
		a.mu.Unlock()
		return "", "Error: Invite expired"
	}

	inviter, inviterExists := a.users[pending.InviterID]
	invitee, inviteeExists := a.users[pending.InviteeID]
	if !inviterExists || !inviteeExists {
		delete(a.pendingInvites, inviteID)
		a.mu.Unlock()
		return "", "Error: User not found"
	}

	if inviter.RoomID != nil || invitee.RoomID != nil {
		delete(a.pendingInvites, inviteID)
		a.mu.Unlock()
		return "", "Error: One of the users is already in a room"
	}

	a.roomCounter++
	roomID := fmt.Sprintf("room_%d", a.roomCounter)
	room := &Room{
		ID:              roomID,
		Name:            fmt.Sprintf("Room %d", a.roomCounter),
		OwnerID:         inviter.ID, // Inviter becomes the owner
		UserIDs:         []string{},
		ApprovedUserIDs: []string{inviter.ID, invitee.ID},
	}
	a.rooms[roomID] = room
	delete(a.pendingInvites, inviteID)
	a.mu.Unlock()

	_, err := a.JoinRoom(inviter.ID, roomID)
	if err != nil {
		return "", err.Error()
	}

	_, err = a.JoinRoom(invitee.ID, roomID)
	if err != nil {
		return "", err.Error()
	}

	return roomID, fmt.Sprintf("Room %s ready for collaboration", roomID)
}

func (a *App) emitClipboardPermissionEvent(granted bool, message string) {
	if a.ctx == nil {
		return
	}

	payload := map[string]interface{}{
		"granted": granted,
	}
	if message != "" {
		payload["message"] = message
	}

	runtime.EventsEmit(a.ctx, wailsEventClipboardPermission, payload)
}

func (a *App) emitClipboardButtonEvent(screenX, screenY int) {
	if a.ctx == nil {
		return
	}

	runtime.EventsEmit(a.ctx, wailsEventClipboardShowButton, map[string]int{
		"screenX": screenX,
		"screenY": screenY,
	})
}

// CreateUser creates a new user in the system
func (a *App) CreateUser(name string) *User {
	a.mu.Lock()
	defer a.mu.Unlock()

	cleanName := sanitizeUserName(name)
	if cleanName == "" {
		cleanName = fmt.Sprintf("User %d", a.userCounter+1)
	}

	// Check for name conflict and resolve it
	originalName := cleanName
	counter := 1
	for {
		conflict := false
		for _, u := range a.users {
			if u.Name == cleanName {
				conflict = true
				break
			}
		}
		if !conflict {
			break
		}
		cleanName = fmt.Sprintf("%s (%d)", originalName, counter)
		counter++
	}

	// Generate user ID based on monotonic counter
	a.userCounter++
	userID := fmt.Sprintf("user_%d", a.userCounter)
	user := &User{
		ID:       userID,
		Name:     cleanName,
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

// startCleanupTasks starts background cleanup goroutines for memory management
func (a *App) startCleanupTasks(ctx context.Context) {
	// Room cleanup ticker
	roomTicker := time.NewTicker(roomCleanupInterval)
	go func() {
		defer roomTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-roomTicker.C:
				a.cleanupEmptyRooms()
			}
		}
	}()

	inviteTicker := time.NewTicker(10 * time.Second)
	go func() {
		defer inviteTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-inviteTicker.C:
				a.cleanupExpiredInvites()
			}
		}
	}()

	fmt.Printf("Started cleanup tasks: room cleanup every %v, invite cleanup every 10s\n", roomCleanupInterval)
}

// cleanupEmptyRooms removes rooms with no active users
func (a *App) cleanupEmptyRooms() {
	a.mu.Lock()
	defer a.mu.Unlock()

	roomsToDelete := make([]string, 0)

	for roomID, room := range a.rooms {
		if len(room.UserIDs) == 0 {
			roomsToDelete = append(roomsToDelete, roomID)
		}
	}

	for _, roomID := range roomsToDelete {
		delete(a.rooms, roomID)
		fmt.Printf("Cleaned up empty room: %s\n", roomID)
	}

	if len(roomsToDelete) > 0 {
		fmt.Printf("Cleanup completed: removed %d empty rooms\n", len(roomsToDelete))
	}
}

func (a *App) cleanupExpiredInvites() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.pendingInvites) == 0 {
		return
	}

	now := time.Now()
	for id, invite := range a.pendingInvites {
		if now.After(invite.ExpiresAt) {
			delete(a.pendingInvites, id)
		}
	}
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
	remainingMembers := append([]string{}, room.UserIDs...)

	// If the leaver was the owner, assign a new owner
	if room.OwnerID == user.ID && len(remainingMembers) > 0 {
		room.OwnerID = remainingMembers[0]
		fmt.Printf("Room %s owner changed to %s\n", room.ID, room.OwnerID)
	}

	leavePayload := map[string]interface{}{
		"roomId":   room.ID,
		"roomName": room.Name,
		"userId":   user.ID,
		"userName": user.Name,
		"ownerId":  room.OwnerID,
	}
	a.sseManager.BroadcastToUsers(remainingMembers, EventUserLeft, leavePayload, "")

	// If room has less than 2 users, delete it
	if len(room.UserIDs) < 2 {
		delete(a.rooms, room.ID)
		// Remove room reference from remaining users
		for _, uid := range remainingMembers {
			if u, exists := a.users[uid]; exists {
				u.RoomID = nil
			}
		}
		// If this was the current room, clear it
		if a.currentRoom != nil && a.currentRoom.ID == room.ID {
			a.currentRoom = nil
		}

		if len(remainingMembers) > 0 {
			roomPayload := map[string]interface{}{
				"roomId":   room.ID,
				"roomName": room.Name,
			}
			a.sseManager.BroadcastToUsers(remainingMembers, EventRoomDeleted, roomPayload, "")
		}
		return fmt.Sprintf("Room %s deleted: insufficient users after %s left", room.Name, user.Name)
	}

	return fmt.Sprintf("%s left room %s", user.Name, room.Name)
}

// JoinRoom adds a user to a room and notifies all room members
func (a *App) JoinRoom(userID, roomID string) (*Room, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	user, exists := a.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	room, exists := a.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room not found")
	}

	// Check permissions for non-host rooms
	if room.OwnerID != "" && room.OwnerID != "host" {
		isApproved := false
		if room.OwnerID == userID {
			isApproved = true
		} else {
			for _, id := range room.ApprovedUserIDs {
				if id == userID {
					isApproved = true
					break
				}
			}
		}
		if !isApproved {
			return nil, fmt.Errorf("permission denied: join request required")
		}
	}

	// Check if user is already in the room
	if contains(room.UserIDs, userID) {
		fmt.Printf("User %s already in room %s\n", userID, roomID)
		return room, nil
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

	return room, nil
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

	safeMessage := sanitizeChatMessage(message)
	if safeMessage == "" {
		return "Error: Message cannot be empty"
	}

	// Create chat message
	msg := &ChatMessage{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()), // unique ID
		RoomID:    roomID,
		UserID:    userID,
		UserName:  userName,
		Message:   safeMessage,
		Timestamp: time.Now().Unix(),
	}

	// Create item
	item := &Item{
		ID:   msg.ID,
		Type: ItemChat,
		Data: msg,
	}

	// Add operation and notify consumers about the delta (not the entire history)
	a.historyPool.AddOperation(roomID, OpAdd, msg.ID, item, userID, userName)
	// Broadcast to all members including the sender (pass empty string as excludeUserID)
	a.sseManager.BroadcastToUsers(members, EventChatMessage, msg, "")

	fmt.Printf("Chat message from %s in room %s: %s\n", userName, roomID, safeMessage)
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
	http.HandleFunc("/api/invite/accept", a.handleAcceptInvite)
	http.HandleFunc("/api/join", a.handleJoinRoom)
	http.HandleFunc("/api/chat", a.handleChat)
	http.HandleFunc("/api/chat/", a.handleChat)
	http.HandleFunc("/api/operations/", a.handleOperations)
	http.HandleFunc("/api/join/request", a.handleJoinRequest)
	http.HandleFunc("/api/join/approve", a.handleApproveJoin)
	http.HandleFunc("/api/download/", a.handleDownload)
	http.HandleFunc("/api/clipboard", a.handleClipboardUpload)
	http.HandleFunc("/api/clipboard/", a.handleZipUpload)
	http.HandleFunc("/api/leave", a.handleLeave)
	http.HandleFunc("/api/sse", a.handleSSE)

	fmt.Printf("Starting HTTP server on port %s\n", port)
	go http.ListenAndServe(":"+port, nil)
}

// RequestJoinRoom handles a user requesting to join a room
func (a *App) RequestJoinRoom(userID, roomID string) (string, error) {
	a.mu.RLock()
	user, userExists := a.users[userID]
	room, roomExists := a.rooms[roomID]
	a.mu.RUnlock()

	if !userExists {
		return "", fmt.Errorf("user not found")
	}
	if !roomExists {
		return "", fmt.Errorf("room not found")
	}

	// If user is already in the room, just return success
	if contains(room.UserIDs, userID) {
		return "Already in room", nil
	}

	// Notify owner
	payload := map[string]interface{}{
		"roomId":        room.ID,
		"roomName":      room.Name,
		"requesterId":   user.ID,
		"requesterName": user.Name,
	}

	fmt.Printf("Sending join request from %s to owner %s of room %s\n", user.Name, room.OwnerID, room.Name)
	if err := a.sseManager.SendToClient(room.OwnerID, EventJoinRequest, payload); err != nil {
		return "", fmt.Errorf("failed to notify room owner: %v", err)
	}

	return "Request sent to room owner", nil
}

// ApproveJoinRequest handles the owner approving a join request
func (a *App) ApproveJoinRequest(ownerID, requesterID, roomID string) error {
	a.mu.Lock()
	room, exists := a.rooms[roomID]
	if !exists {
		a.mu.Unlock()
		return fmt.Errorf("room not found")
	}

	if room.OwnerID != ownerID {
		a.mu.Unlock()
		return fmt.Errorf("permission denied: not room owner")
	}

	// Add to approved list
	if !contains(room.ApprovedUserIDs, requesterID) {
		room.ApprovedUserIDs = append(room.ApprovedUserIDs, requesterID)
	}
	a.mu.Unlock()

	_, err := a.JoinRoom(requesterID, roomID)
	return err
}
