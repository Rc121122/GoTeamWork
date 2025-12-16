package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
				http.Error(w, "Username already exists", http.StatusConflict)
				return
			}
		}
		a.mu.RUnlock()

		user := a.CreateUser(sanitizedName)
		token, err := a.issueToken(user.ID)
		if err != nil {
			http.Error(w, "Failed to issue token", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateUserResponse{User: user, Token: token})
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

	authUser, err := a.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract user ID from URL path
	path := r.URL.Path
	userID := ""
	if len(path) > len("/api/users/") {
		userID = path[len("/api/users/"):]
	}

	if _, err := enforceUserMatch(userID, authUser); err != nil {
		http.Error(w, "Forbidden: userId does not match token", http.StatusForbidden)
		return
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

	authUser, err := a.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

		room := a.CreateRoom(roomName, authUser.ID)
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

	authUser, err := a.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	inviterID, err := enforceUserMatch(req.InviterID, authUser)
	if err != nil {
		http.Error(w, "Forbidden: userId does not match token", http.StatusForbidden)
		return
	}
	req.InviterID = inviterID

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

	authUser, err := a.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	inviteeID, err := enforceUserMatch(req.InviteeID, authUser)
	if err != nil {
		http.Error(w, "Forbidden: userId does not match token", http.StatusForbidden)
		return
	}
	req.InviteeID = inviteeID

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

	authUser, err := a.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract room ID from URL path
	path := r.URL.Path
	roomID := ""
	if len(path) > len("/api/chat/") {
		roomID = path[len("/api/chat/"):]
	}

	if r.Method == "GET" && roomID != "" {
		if authUser.RoomID == nil || *authUser.RoomID != roomID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
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

		reqUserID, err := enforceUserMatch(req.UserID, authUser)
		if err != nil {
			http.Error(w, "Forbidden: userId does not match token", http.StatusForbidden)
			return
		}
		req.UserID = reqUserID

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

	authUser, err := a.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req JoinRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	reqUserID, err := enforceUserMatch(req.UserID, authUser)
	if err != nil {
		http.Error(w, "Forbidden: userId does not match token", http.StatusForbidden)
		return
	}
	req.UserID = reqUserID

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

// handleJoinRequest handles POST /api/join/request
func (a *App) handleJoinRequest(w http.ResponseWriter, r *http.Request) {
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

	authUser, err := a.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req JoinRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	reqUserID, err := enforceUserMatch(req.UserID, authUser)
	if err != nil {
		http.Error(w, "Forbidden: userId does not match token", http.StatusForbidden)
		return
	}
	req.UserID = reqUserID

	msg, err := a.RequestJoinRoom(req.UserID, req.RoomID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := APIResponse{Message: msg}
	json.NewEncoder(w).Encode(response)
}

// handleApproveJoin handles POST /api/join/approve
func (a *App) handleApproveJoin(w http.ResponseWriter, r *http.Request) {
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

	authUser, err := a.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	type ApproveRequest struct {
		OwnerID     string `json:"ownerId"`
		RequesterID string `json:"requesterId"`
		RoomID      string `json:"roomId"`
	}

	var req ApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ownerID, err := enforceUserMatch(req.OwnerID, authUser)
	if err != nil {
		http.Error(w, "Forbidden: userId does not match token", http.StatusForbidden)
		return
	}
	req.OwnerID = ownerID

	err = a.ApproveJoinRequest(req.OwnerID, req.RequesterID, req.RoomID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := APIResponse{Message: "Approved"}
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

	authUser, err := a.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req LeaveRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	reqUserID, err := enforceUserMatch(req.UserID, authUser)
	if err != nil {
		http.Error(w, "Forbidden: userId does not match token", http.StatusForbidden)
		return
	}
	req.UserID = reqUserID

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

	authUser, err := a.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	if !a.userInRoom(authUser.ID, roomID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	sinceID := strings.TrimSpace(r.URL.Query().Get("since"))
	sinceHash := strings.TrimSpace(r.URL.Query().Get("sinceHash"))

	operations := a.GetOperations(roomID, sinceID, sinceHash)
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

	// Find the operation in any room
	var targetOp *Operation

	a.mu.RLock()
	roomIDs := make([]string, 0, len(a.rooms))
	for id := range a.rooms {
		roomIDs = append(roomIDs, id)
	}
	a.mu.RUnlock()

	for _, rid := range roomIDs {
		ops := a.GetOperations(rid, "", "")
		for _, op := range ops {
			if op.ID == opID {
				targetOp = op
				break
			}
		}
		if targetOp != nil {
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
	if !ok || itemData.Type != clip_helper.ClipboardFile {
		http.Error(w, "No file data available", http.StatusNotFound)
		return
	}

	if itemData.IsSingleFile {
		if len(itemData.SingleFileData) == 0 {
			http.Error(w, "No file data available", http.StatusNotFound)
			return
		}
		filename := itemData.SingleFileName
		if filename == "" {
			filename = fmt.Sprintf("shared_file_%s", opID)
		}
		mimeType := itemData.SingleFileMime
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(itemData.SingleFileData)))
		w.Write(itemData.SingleFileData)
		return
	}

	if len(itemData.ZipData) == 0 {
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
	a.mu.RLock()
	user, exists := a.users[req.UserID]
	a.mu.RUnlock()

	if !exists {
		http.Error(w, "User not found", http.StatusForbidden)
		return
	}

	if user.RoomID == nil {
		http.Error(w, "User not in a room", http.StatusForbidden)
		return
	}
	roomID := *user.RoomID

	itemID := fmt.Sprintf("clip_%d", time.Now().UnixNano())
	histItem := &Item{
		ID:   itemID,
		Type: ItemClipboard,
		Data: &req.Item,
	}

	fmt.Printf("Received clipboard upload from %s in room %s: %d files\n", req.UserName, roomID, len(req.Item.Files))

	op := a.historyPool.AddOperation(roomID, OpAdd, itemID, histItem, req.UserID, req.UserName)

	// Get room members for broadcast
	a.mu.RLock()
	room, roomExists := a.rooms[roomID]
	var members []string
	if roomExists {
		members = append(members, room.UserIDs...)
	}
	a.mu.RUnlock()

	if roomExists {
		a.sseManager.BroadcastToUsers(members, EventClipboardCopied, op, "")
	}

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

	isSingle := r.URL.Query().Get("single") == "1"
	fileName := r.Header.Get("X-Clipboard-File-Name")
	fileMime := r.Header.Get("X-Clipboard-File-Mime")
	fileThumb := r.Header.Get("X-Clipboard-File-Thumb")
	fileSizeHeader := r.Header.Get("X-Clipboard-File-Size")

	fmt.Printf("Received %s upload for op: %s\n", map[bool]string{true: "single file", false: "zip"}[isSingle], opID)

	// Limit payload size to 10GB
	const maxDataSize = 10 * 1024 * 1024 * 1024            // 10GB
	limitedReader := io.LimitReader(r.Body, maxDataSize+1) // +1 to detect if over limit

	zipData, err := io.ReadAll(limitedReader)
	if err != nil {
		fmt.Printf("Failed to read zip body: %v\n", err)
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}

	if len(zipData) > maxDataSize {
		fmt.Printf("Upload size %d bytes exceeds limit of %d bytes\n", len(zipData), maxDataSize)
		http.Error(w, "File too large (max 10GB)", http.StatusRequestEntityTooLarge)
		return
	}

	fmt.Printf("Zip data size: %d bytes\n", len(zipData))

	// Update item in history pool
	// Find operation in any room
	var targetOp *Operation
	var roomID string

	a.mu.RLock()
	roomIDs := make([]string, 0, len(a.rooms))
	for id := range a.rooms {
		roomIDs = append(roomIDs, id)
	}
	a.mu.RUnlock()

	for _, rid := range roomIDs {
		ops := a.GetOperations(rid, "", "")
		for _, op := range ops {
			if op.ID == opID {
				targetOp = op
				roomID = rid
				break
			}
		}
		if targetOp != nil {
			break
		}
	}

	if targetOp == nil {
		fmt.Printf("Operation not found: %s\n", opID)
		http.Error(w, "Operation not found", http.StatusNotFound)
		return
	}

	if roomID != "" && !a.userInRoom(targetOp.UserID, roomID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
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
	if isSingle {
		if fileName == "" {
			fileName = fmt.Sprintf("shared_file_%s", opID)
		}
		if fileMime == "" {
			fileMime = "application/octet-stream"
		}
		itemData.IsSingleFile = true
		itemData.SingleFileName = fileName
		itemData.SingleFileMime = fileMime
		if fileThumb != "" {
			itemData.SingleFileThumb = fileThumb
		}
		if fileSizeHeader != "" {
			if parsed, err := strconv.ParseInt(fileSizeHeader, 10, 64); err == nil {
				itemData.SingleFileSize = parsed
			} else {
				itemData.SingleFileSize = int64(len(zipData))
			}
		} else {
			itemData.SingleFileSize = int64(len(zipData))
		}
		itemData.SingleFileData = zipData
		itemData.ZipData = nil
		itemData.Text = fmt.Sprintf("%s (%s) ready", fileName, clip_helper.HumanFileSize(itemData.SingleFileSize))
	} else {
		itemData.ZipData = zipData
		itemData.IsSingleFile = false
		itemData.SingleFileData = nil
		itemData.Text = fmt.Sprintf("%d files compressed (ready)", len(itemData.Files))
	}
	a.mu.Unlock()

	fmt.Printf("Updated operation %s with zip data. Text: %s\n", opID, itemData.Text)

	// Broadcast to room members
	if roomID != "" {
		a.mu.RLock()
		room, roomExists := a.rooms[roomID]
		var members []string
		if roomExists {
			members = append(members, room.UserIDs...)
		}
		a.mu.RUnlock()

		if roomExists {
			a.sseManager.BroadcastToUsers(members, EventClipboardUpdated, targetOp, "")
		}
	}

	w.WriteHeader(http.StatusOK)
}
