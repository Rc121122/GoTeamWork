package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"GOproject/clip_helper"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grandcat/zeroconf"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	maxOperationsPerRoom   = 1000             // Maximum operations to keep per room
	maxChatMessagesPerRoom = 100              // Maximum chat messages to keep per room
	roomCleanupInterval    = 30 * time.Minute // Check for empty rooms every 30 minutes
	userTimeout            = 24 * time.Hour   // Remove inactive users after 24 hours
	inviteTimeout          = 30 * time.Second // Pending invites expire after 30 seconds
	defaultJWTExpiry       = 24 * time.Hour
)

// JWTClaims captures the authenticated user ID for HMAC tokens.
type JWTClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

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
	var parentHash string
	if ops, exists := hp.operations[roomID]; exists && len(ops) > 0 {
		last := ops[len(ops)-1]
		parentID = last.ID
		parentHash = last.Hash
	}

	timestamp := time.Now().Unix()
	hash := computeOperationHash(parentHash, opType, itemID, item, userID, userName, timestamp)

	op := &Operation{
		ID:         id,
		ParentID:   parentID,
		ParentHash: parentHash,
		Hash:       hash,
		OpType:     opType,
		ItemID:     itemID,
		Item:       item,
		Timestamp:  timestamp,
		UserID:     userID,
		UserName:   userName,
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

// computeOperationHash builds a stable hash for an operation to support incremental sync.
func computeOperationHash(parentHash string, opType OperationType, itemID string, item *Item, userID, userName string, timestamp int64) string {
	fingerprint := buildItemFingerprint(item)

	payload := struct {
		ParentHash string        `json:"parentHash"`
		OpType     OperationType `json:"opType"`
		ItemID     string        `json:"itemId"`
		Item       interface{}   `json:"item"`
		UserID     string        `json:"userId"`
		UserName   string        `json:"userName"`
		Timestamp  int64         `json:"timestamp"`
	}{
		ParentHash: parentHash,
		OpType:     opType,
		ItemID:     itemID,
		Item:       fingerprint,
		UserID:     userID,
		UserName:   userName,
		Timestamp:  timestamp,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		// Fallback to a timestamp-only hash to avoid breaking flows in unlikely marshal failures
		h := sha256.Sum256([]byte(fmt.Sprintf("fallback-%d", timestamp)))
		return hex.EncodeToString(h[:])
	}

	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// buildItemFingerprint produces a lightweight, deterministic view of an item for hashing.
func buildItemFingerprint(item *Item) interface{} {
	if item == nil {
		return nil
	}

	base := struct {
		ID   string   `json:"id"`
		Type ItemType `json:"type"`
	}{
		ID:   item.ID,
		Type: item.Type,
	}

	switch v := item.Data.(type) {
	case *ChatMessage:
		return struct {
			Base      interface{} `json:"base"`
			Message   string      `json:"message"`
			Timestamp int64       `json:"timestamp"`
		}{
			Base:      base,
			Message:   v.Message,
			Timestamp: v.Timestamp,
		}
	case *clip_helper.ClipboardItem:
		return struct {
			Base       interface{} `json:"base"`
			Text       string      `json:"text"`
			FileCount  int         `json:"fileCount"`
			ZipBytes   int         `json:"zipBytes"`
			ImageBytes int         `json:"imageBytes"`
		}{
			Base:       base,
			Text:       v.Text,
			FileCount:  len(v.Files),
			ZipBytes:   len(v.ZipData),
			ImageBytes: len(v.Image),
		}
	default:
		return base
	}
}

// GetOperations returns operations for a room after a given operation ID or hash (hash preferred).
func (hp *HistoryPool) GetOperations(roomID, sinceID, sinceHash string) []*Operation {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	ops := hp.operations[roomID]
	if ops == nil {
		return []*Operation{}
	}

	startIdx := 0

	if sinceHash != "" {
		startIdx = -1
		for i, op := range ops {
			if op.Hash == sinceHash {
				startIdx = i + 1
				break
			}
		}
		if startIdx == -1 {
			// Unknown hash (likely trimmed); return full list so the client can resync
			startIdx = 0
		}
	} else if sinceID != "" {
		startIdx = -1
		for i, op := range ops {
			if op.ID == sinceID {
				startIdx = i + 1
				break
			}
		}
		if startIdx == -1 {
			return []*Operation{}
		}
	}

	if startIdx >= len(ops) {
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

	jwtSecret      []byte
	zeroconfServer *zeroconf.Server
}

const (
	wailsEventClipboardShowButton = "clipboard:show-share-button"
	wailsEventClipboardPermission = "clipboard:permission-state"
)

// NewApp creates a new App application struct
func NewApp(mode string) *App {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		if generated, err := generateJWTSecret(); err == nil {
			secret = generated
			if mode == "host" {
				fmt.Println("Generated ephemeral JWT secret for host mode; set JWT_SECRET to persist across restarts")
			}
		} else {
			fmt.Printf("WARNING: failed to generate JWT secret, using insecure default: %v\n", err)
			secret = "dev-insecure-change-me"
		}
	}

	app := &App{
		Mode:           mode,
		users:          make(map[string]*User),
		rooms:          make(map[string]*Room),
		pendingInvites: make(map[string]*PendingInvite),
		historyPool:    NewHistoryPool(),
		sseManager:     NewSSEManager(),
		jwtSecret:      []byte(secret),
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

		// Register zeroconf service for discovery
		server, err := zeroconf.Register("GoTeamWork", "_http._tcp", "local.", 8080, []string{"version=1.0"}, nil)
		if err != nil {
			fmt.Printf("Failed to register zeroconf service: %v\n", err)
		} else {
			a.zeroconfServer = server
			fmt.Println("Zeroconf service registered for discovery")
		}

		// Start cleanup goroutines for host mode
		go a.startCleanupTasks(ctx)
	} else if a.Mode == "client" {
		// Initialize network client for client mode (URL will be set when user connects)
		a.networkClient = NewNetworkClient("")
		fmt.Println("Client mode initialized. Please enter server address to connect.")
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

// CreateRoom creates a new room with the given name and owner
func (a *App) CreateRoom(name, ownerID string) *Room {
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
		OwnerID: ownerID,
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

	// If inviter is already in a room, use that room for the invite; otherwise a new room will be created on accept.
	var targetRoomID string
	var targetRoomName string
	if inviter.RoomID != nil {
		room, ok := a.rooms[*inviter.RoomID]
		if !ok {
			return "", "Error: Inviter's room not found", 0
		}
		targetRoomID = room.ID
		targetRoomName = room.Name
	}

	if invitee.RoomID != nil {
		if targetRoomID != "" && *invitee.RoomID == targetRoomID {
			return "", "Error: Invitee already in this room", 0
		}
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
		RoomID:    targetRoomID,
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
	if targetRoomID != "" {
		payload["roomId"] = targetRoomID
		payload["roomName"] = targetRoomName
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

func (a *App) userInRoom(userID, roomID string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	room, ok := a.rooms[roomID]
	if !ok {
		return false
	}

	return contains(room.UserIDs, userID)
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

	// If this invite targets an existing room, join that room instead of creating a new one.
	if pending.RoomID != "" {
		if inviter.RoomID == nil || *inviter.RoomID != pending.RoomID {
			delete(a.pendingInvites, inviteID)
			a.mu.Unlock()
			return "", "Error: Inviter no longer in the room"
		}

		if invitee.RoomID != nil {
			delete(a.pendingInvites, inviteID)
			a.mu.Unlock()
			return "", "Error: Invitee already in a room"
		}

		room, ok := a.rooms[pending.RoomID]
		if !ok {
			delete(a.pendingInvites, inviteID)
			a.mu.Unlock()
			return "", "Error: Room not found"
		}

		if room.OwnerID != "" && room.OwnerID != "host" && !contains(room.ApprovedUserIDs, inviteeID) {
			room.ApprovedUserIDs = append(room.ApprovedUserIDs, inviteeID)
		}

		delete(a.pendingInvites, inviteID)
		a.mu.Unlock()

		_, err := a.JoinRoom(inviteeID, pending.RoomID)
		if err != nil {
			return "", err.Error()
		}

		return pending.RoomID, fmt.Sprintf("Room %s joined via invite", room.Name)
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

// SetServerURL sets the server URL for the network client (for client mode)
// Supports: localhost, LAN IPs (192.168.x.x), and Cloudflare tunnels (https://xxx.trycloudflare.com)
func (a *App) SetServerURL(url string) {
	if a.Mode != "client" || a.networkClient == nil {
		fmt.Println("SetServerURL: Not in client mode or network client not initialized")
		return
	}

	// URL should already be properly formatted from frontend
	// Just set it directly
	a.networkClient.serverURL = url
	fmt.Printf("Server URL set to: %s\n", url)

	// Don't auto-connect here - connection will happen when user creates account
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
	// Broadcast to all members except the sender
	a.sseManager.BroadcastToUsers(members, EventChatMessage, msg, userID)

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

// GetOperations returns operations for a room since a given ID or hash.
func (a *App) GetOperations(roomID, sinceID, sinceHash string) []*Operation {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// For global clipboard, allow roomID == "global"
	if roomID != "global" {
		if _, exists := a.rooms[roomID]; !exists {
			return []*Operation{}
		}
	}

	return a.historyPool.GetOperations(roomID, sinceID, sinceHash)
}

func (a *App) issueToken(userID string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(defaultJWTExpiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

func (a *App) authenticateRequest(r *http.Request) (*User, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return nil, errors.New("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, errors.New("invalid Authorization header")
	}

	tokenString := strings.TrimSpace(parts[1])
	if tokenString == "" {
		return nil, errors.New("empty token")
	}

	claims := &JWTClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.jwtSecret, nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}

	a.mu.RLock()
	user, ok := a.users[claims.UserID]
	a.mu.RUnlock()
	if !ok {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func (a *App) authenticateToken(tokenString string) (*User, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, errors.New("empty token")
	}

	claims := &JWTClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.jwtSecret, nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}

	a.mu.RLock()
	user, ok := a.users[claims.UserID]
	a.mu.RUnlock()
	if !ok {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func enforceUserMatch(reqUserID string, authUser *User) (string, error) {
	if reqUserID == "" {
		return authUser.ID, nil
	}
	if reqUserID != authUser.ID {
		return "", errors.New("userId does not match token")
	}
	return reqUserID, nil
}

// corsMiddleware wraps a handler with CORS headers
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all requests
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func getEnvDefault(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func generateJWTSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// StartHTTPServer starts the HTTP server for REST API
func (a *App) StartHTTPServer(port string) {
	http.HandleFunc("/api/users", corsMiddleware(a.handleUsers))
	http.HandleFunc("/api/users/", corsMiddleware(a.handleUserByID))
	http.HandleFunc("/api/rooms", corsMiddleware(a.handleRooms))
	http.HandleFunc("/api/invite", corsMiddleware(a.handleInvite))
	http.HandleFunc("/api/invite/accept", corsMiddleware(a.handleAcceptInvite))
	http.HandleFunc("/api/join", corsMiddleware(a.handleJoinRoom))
	http.HandleFunc("/api/chat", corsMiddleware(a.handleChat))
	http.HandleFunc("/api/chat/", corsMiddleware(a.handleChat))
	http.HandleFunc("/api/operations/", corsMiddleware(a.handleOperations))
	http.HandleFunc("/api/join/request", corsMiddleware(a.handleJoinRequest))
	http.HandleFunc("/api/join/approve", corsMiddleware(a.handleApproveJoin))
	http.HandleFunc("/api/download/", corsMiddleware(a.handleDownload))
	http.HandleFunc("/api/clipboard", corsMiddleware(a.handleClipboardUpload))
	http.HandleFunc("/api/clipboard/", corsMiddleware(a.handleZipUpload))
	http.HandleFunc("/api/leave", corsMiddleware(a.handleLeave))
	http.HandleFunc("/api/sse", corsMiddleware(a.handleSSE))

	fmt.Printf("Starting HTTP server on port %s\n", port)
	listener, err := net.Listen("tcp4", "0.0.0.0:"+port)
	if err != nil {
		fmt.Printf("Failed to listen on port %s: %v\n", port, err)
		return
	}
	go http.Serve(listener, nil)
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
