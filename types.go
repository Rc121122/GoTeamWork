package main

import "sync"

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

// HTTP Request/Response Types
type CreateUserRequest struct {
	Name string `json:"name"`
}

type InviteUserRequest struct {
	UserID string `json:"userId"`
}

type ChatMessageRequest struct {
	RoomID  string `json:"roomId"`
	UserID  string `json:"userId"`
	Message string `json:"message"`
}

type LeaveRoomRequest struct {
	UserID string `json:"userId"`
}

type CreateRoomRequest struct {
	Name string `json:"name"`
}

type APIResponse struct {
	Message string `json:"message"`
}