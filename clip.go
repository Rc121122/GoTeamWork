package main

import (
	"fmt"

	"golang.design/x/clipboard"
)

// ClipboardItemType represents the type of clipboard content
type ClipboardItemType string

const (
	ClipboardText  ClipboardItemType = "text"
	ClipboardImage ClipboardItemType = "image"
)

// ClipboardItem represents a clipboard item with its content
type ClipboardItem struct {
	Type     ClipboardItemType `json:"type"`
	Text     string            `json:"text,omitempty"`
	Image    []byte            `json:"image,omitempty"` // PNG encoded
}

// ReadClipboard reads the current clipboard content and returns a ClipboardItem
func ReadClipboard() (*ClipboardItem, error) {
	// Initialize clipboard if needed
	err := clipboard.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize clipboard: %w", err)
	}

	// Try to read as image
	if imgData := clipboard.Read(clipboard.FmtImage); len(imgData) > 0 {
		return &ClipboardItem{
			Type:  ClipboardImage,
			Image: imgData,
		}, nil
	}

	// Try to read as text
	if textData := clipboard.Read(clipboard.FmtText); len(textData) > 0 {
		text := string(textData)
		return &ClipboardItem{
			Type: ClipboardText,
			Text: text,
		}, nil
	}

	return nil, fmt.Errorf("no supported clipboard content found")
}

// StartClipboardMonitor starts monitoring for the copy hotkey (Ctrl+Shift+C / Cmd+Shift+C)
func (a *App) StartClipboardMonitor() {
	// TODO: Implement cross-platform global hotkey detection
	// For now, this is a placeholder
	fmt.Println("Clipboard monitor started (hotkey detection not implemented yet)")
}

// handleClipboardCopy processes a copied clipboard item
func (a *App) handleClipboardCopy(item *ClipboardItem) {
	fmt.Printf("Clipboard copied: type=%s\n", item.Type)

	// For now, just broadcast to all connected clients
	// Later, we can store it in room clipboard or something
	a.sseManager.BroadcastToAll(EventClipboardCopied, item)
}

// GetClipboardItem is a Wails-exposed function to manually get clipboard content
func (a *App) GetClipboardItem() (*ClipboardItem, error) {
	return ReadClipboard()
}

