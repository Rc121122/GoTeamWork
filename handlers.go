package main

import (
	"encoding/json"
	"net/http"
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

		room := a.CreateRoom(req.Name)
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

	roomID, result := a.InviteWithRoom(req.UserID, req.InviterID)
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

	result := a.JoinRoom(req.UserID, req.RoomID)
	response := APIResponse{Message: result, RoomID: req.RoomID}
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

	sinceID := r.URL.Query().Get("since")

	operations := a.GetOperations(roomID, sinceID)
	json.NewEncoder(w).Encode(operations)
}
