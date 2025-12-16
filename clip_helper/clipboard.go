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
	ZipData []byte            `json:"-"`               // Zip file content
	Files   []string          `json:"files,omitempty"` // File paths

	IsSingleFile    bool   `json:"isSingleFile,omitempty"`
	SingleFileName  string `json:"singleFileName,omitempty"`
	SingleFileMime  string `json:"singleFileMime,omitempty"`
	SingleFileSize  int64  `json:"singleFileSize,omitempty"`
	SingleFileThumb string `json:"singleFileThumb,omitempty"`
	SingleFileData  []byte `json:"-"` // raw bytes for direct download

	ZipFilePath     string `json:"-"` // Path to zip file on disk
	SingleFilePath  string `json:"-"` // Path to single file on disk
}

const (
	ClipboardShareCooldown = 250 * time.Millisecond
	ClipboardCacheTTL      = 8 * time.Second
)
