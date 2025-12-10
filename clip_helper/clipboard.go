package clip_helper

import (
	"time"
)

// ClipboardItemType represents the type of clipboard content
type ClipboardItemType string

const (
	ClipboardText  ClipboardItemType = "text"
	ClipboardImage ClipboardItemType = "image"
	ClipboardFile  ClipboardItemType = "file" // New type for files/zip
)

// ClipboardItem represents a clipboard item with its content
type ClipboardItem struct {
	Type    ClipboardItemType `json:"type"`
	Text    string            `json:"text,omitempty"`
	Image   []byte            `json:"image,omitempty"` // PNG encoded
	ZipData []byte            `json:"-"` // Zip file content
	Files   []string          `json:"files,omitempty"` // File paths
}

const (
	ClipboardShareCooldown = 250 * time.Millisecond
	ClipboardCacheTTL      = 8 * time.Second
)
