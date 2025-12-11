package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"GOproject/clip_helper"
)

// handleUsers handles GET /api/users (list all users) and POST /api/users (create user)
func (a *App) handleUsers(w http.ResponseWriter, r *http.Request) {
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
		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		sanitizedName := sanitizeUserName(req.Name)
		if sanitizedName == "" {
			http.Error(w, "Name is invalid", http.StatusBadRequest)
			return
		}

		// Check if username is unique
		a.mu.RLock()
		for _, user := range a.users {
			if user.Name == sanitizedName {
				a.mu.RUnlock()
				// Return existing user instead of error to allow reconnection
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(user)
				return
			}
		}
		a.mu.RUnlock()

		user := a.CreateUser(sanitizedName)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleUserByID handles GET /api/users/{id} (get specific user)
func (a *App) handleUserByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL path
	path := r.URL.Path
	userID := ""
	if len(path) > len("/api/users/") {
		userID = path[len("/api/users/"):]
	}

	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	a.mu.RLock()
	user, exists := a.users[userID]
	a.mu.RUnlock()

	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(user)
}

// handleRooms handles GET /api/rooms (list rooms) and POST /api/rooms (create room)
func (a *App) handleRooms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method == "GET" {
		rooms := a.GetAllRooms()
		json.NewEncoder(w).Encode(rooms)
		return
	}

	if r.Method == "POST" {
		var req CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Room name is required", http.StatusBadRequest)
			return
		}

		roomName := sanitizeRoomName(req.Name)
		if roomName == "" {
			http.Error(w, "Room name is invalid", http.StatusBadRequest)
			return
		}

		room := a.CreateRoom(roomName)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(room)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	var req InviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.InviterID == "" {
		http.Error(w, "userId and inviterId are required", http.StatusBadRequest)
		return
	}

	inviteID, result, expiresAt := a.InviteWithRoom(req.UserID, req.InviterID, req.Message)
	response := APIResponse{Message: result, InviteID: inviteID, ExpiresAt: expiresAt}
	json.NewEncoder(w).Encode(response)
}

// handleAcceptInvite handles POST /api/invite/accept
func (a *App) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
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

	var req AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.InviteID == "" || req.InviteeID == "" {
		http.Error(w, "inviteId and inviteeId are required", http.StatusBadRequest)
		return
	}

	roomID, result := a.AcceptInvite(req.InviteID, req.InviteeID)
	status := http.StatusOK
	if strings.HasPrefix(result, "Error") {
		status = http.StatusBadRequest
	}
	w.WriteHeader(status)
	response := APIResponse{Message: result, RoomID: roomID}
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
		var req ChatMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		result := a.SendChatMessage(req.RoomID, req.UserID, req.Message)
		response := APIResponse{Message: result}
		json.NewEncoder(w).Encode(response)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleJoinRoom handles POST /api/join
func (a *App) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
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

	var req JoinRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	room, err := a.JoinRoom(req.UserID, req.RoomID)
	message := ""
	if err != nil {
		message = err.Error()
	} else {
		message = fmt.Sprintf("Joined room %s", room.Name)
	}
	response := APIResponse{Message: message, RoomID: req.RoomID}
	json.NewEncoder(w).Encode(response)
}

// handleLeave handles POST /api/leave
func (a *App) handleLeave(w http.ResponseWriter, r *http.Request) {
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

	var req LeaveRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result := a.LeaveRoom(req.UserID)
	response := APIResponse{Message: result}
	json.NewEncoder(w).Encode(response)
}

// handleOperations handles GET /api/operations/{roomId}?since={operationId}
func (a *App) handleOperations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract room ID from URL path
	path := r.URL.Path
	roomID := ""
	if len(path) > len("/api/operations/") {
		roomID = path[len("/api/operations/"):]
	}

	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	sinceID := strings.TrimSpace(r.URL.Query().Get("since"))

	operations := a.GetOperations(roomID, sinceID)
	json.NewEncoder(w).Encode(operations)
}

// handleDownload handles GET /api/download/{operationId}
func (a *App) handleDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract operation ID from URL path
	path := r.URL.Path
	opID := ""
	if len(path) > len("/api/download/") {
		opID = path[len("/api/download/"):]
	}

	if opID == "" {
		http.Error(w, "Operation ID is required", http.StatusBadRequest)
		return
	}

	// Find the operation in global room (since clipboard is global for now)
	// In a real app, we should probably pass roomID or search all rooms
	ops := a.GetOperations("global", "")
	var targetOp *Operation
	for _, op := range ops {
		if op.ID == opID {
			targetOp = op
			break
		}
	}

	if targetOp == nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if targetOp.Item == nil || targetOp.Item.Type != ItemClipboard {
		http.Error(w, "Invalid item type", http.StatusBadRequest)
		return
	}

	itemData, ok := targetOp.Item.Data.(*clip_helper.ClipboardItem)
	if !ok || itemData.Type != clip_helper.ClipboardFile || len(itemData.ZipData) == 0 {
		http.Error(w, "No file data available", http.StatusNotFound)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"shared_files_%s.zip\"", opID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(itemData.ZipData)))

	w.Write(itemData.ZipData)
}

// handleClipboardUpload handles POST /api/clipboard
func (a *App) handleClipboardUpload(w http.ResponseWriter, r *http.Request) {
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

	var req ClipboardUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Add to history pool
	roomID := "global"
	itemID := fmt.Sprintf("clip_%d", time.Now().UnixNano())
	histItem := &Item{
		ID:   itemID,
		Type: ItemClipboard,
		Data: &req.Item,
	}

	fmt.Printf("Received clipboard upload from %s: %d files\n", req.UserName, len(req.Item.Files))

	op := a.historyPool.AddOperation(roomID, OpAdd, itemID, histItem, req.UserID, req.UserName)
	a.sseManager.BroadcastToAll(EventClipboardCopied, op)

	json.NewEncoder(w).Encode(op)
}

// handleZipUpload handles POST /api/clipboard/{opID}/zip
func (a *App) handleZipUpload(w http.ResponseWriter, r *http.Request) {
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

	path := r.URL.Path
	opID := ""
	if len(path) > len("/api/clipboard/") && strings.Contains(path, "/zip") {
		// /api/clipboard/{opID}/zip
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			opID = parts[3]
		}
	}

	if opID == "" {
		http.Error(w, "Operation ID is required", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received zip upload for op: %s\n", opID)

	zipData, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Failed to read zip body: %v\n", err)
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Zip data size: %d bytes\n", len(zipData))

	// Update item in history pool
	ops := a.GetOperations("global", "")
	var targetOp *Operation
	for _, op := range ops {
		if op.ID == opID {
			targetOp = op
			break
		}
	}

	if targetOp == nil {
		fmt.Printf("Operation not found: %s\n", opID)
		http.Error(w, "Operation not found", http.StatusNotFound)
		return
	}

	if targetOp.Item == nil || targetOp.Item.Type != ItemClipboard {
		http.Error(w, "Invalid item type", http.StatusBadRequest)
		return
	}

	itemData, ok := targetOp.Item.Data.(*clip_helper.ClipboardItem)
	if !ok {
		http.Error(w, "Invalid item data", http.StatusInternalServerError)
		return
	}

	a.mu.Lock()
	itemData.ZipData = zipData
	itemData.Text = fmt.Sprintf("%d files compressed (ready)", len(itemData.Files))
	a.mu.Unlock()

	fmt.Printf("Updated operation %s with zip data\n", opID)

	a.sseManager.BroadcastToAll(EventClipboardUpdated, targetOp)
	w.WriteHeader(http.StatusOK)
}
