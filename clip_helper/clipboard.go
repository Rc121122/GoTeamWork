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
	ZipData []byte            `json:"zip_data,omitempty"` // Zip file content
	// Or maybe just a path if we want to stream it? For now let's keep it simple or use a path.
	// But if we want to share it over network, bytes might be better for small files, or a separate mechanism.
	// The user said "compress filepath(s) to zip when copied".
}

const (
	ClipboardShareCooldown = 250 * time.Millisecond
	ClipboardCacheTTL      = 8 * time.Second
)
