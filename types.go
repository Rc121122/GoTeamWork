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

// OperationType represents the type of operation
type OperationType string

const (
	OpAdd    OperationType = "add"
	OpRemove OperationType = "remove"
	OpModify OperationType = "modify"
)

// ItemType represents the type of item
type ItemType string

const (
	ItemChat      ItemType = "chat"
	ItemClipboard ItemType = "clipboard"
)

// Item represents a data item in the history
type Item struct {
	ID   string      `json:"id"`
	Type ItemType    `json:"type"`
	Data interface{} `json:"data"` // ChatMessage or ClipboardItem
}

// Operation represents a git-style operation on the history
type Operation struct {
	ID        string        `json:"id"`
	ParentID  string        `json:"parentId"`
	OpType    OperationType `json:"opType"`
	ItemID    string        `json:"itemId"`
	Item      *Item         `json:"item,omitempty"`
	Timestamp int64         `json:"timestamp"`
}

// HistoryPool manages operations for all rooms
type HistoryPool struct {
	operations map[string][]*Operation // roomID -> operations
	counter    int
	mu         sync.RWMutex
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