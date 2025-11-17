package main

import (
	"context"
	"fmt"
	"time"

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
	Type  ClipboardItemType `json:"type"`
	Text  string            `json:"text,omitempty"`
	Image []byte            `json:"image,omitempty"` // PNG encoded
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

	// Assume roomID from current room or something, but since broadcast to all, perhaps global or per room.
	// For now, use a default room or broadcast to all rooms.
	// To fit, perhaps add to a global room or modify.

	// For simplicity, since current broadcast to all, and no room context, perhaps use a special roomID like "global"
	roomID := "global" // or get from context

	// Create item ID
	itemID := fmt.Sprintf("clip_%d", time.Now().UnixNano())

	// Create Item
	histItem := &Item{
		ID:   itemID,
		Type: ItemClipboard,
		Data: item,
	}

	// Add operation
	op := a.historyPool.AddOperation(roomID, OpAdd, itemID, histItem)

	// Broadcast to all
	a.sseManager.BroadcastToAll(EventClipboardCopied, op)
}

// addEvent is a wrapper used to detect the hotkey. In production you can
// assign this to robotgo.AddEvents or a similar function. In tests we
// override this variable to simulate hotkey presses.
var addEvent = func(keys ...string) bool {
	// Default: no-op (no hotkey detection) so code compiles on systems
	// without robotgo. Consumers may set this to robotgo.AddEvents.
	return false
}

// StartClipboardHotkey listens for Cmd+Shift+C (macOS) Ctrl + Shift + C (Win/Linux) global hotkey.
// When the hotkey is detected, it reads the system clipboard and invokes cb with the ClipboardItem.
// The function runs until the provided context is cancelled.
func StartClipboardHotkey(ctx context.Context, cb func(*ClipboardItem)) error {
	// Initialize clipboard once
	if err := clipboard.Init(); err != nil {
		return fmt.Errorf("failed to init clipboard: %w", err)
	}

	// Hotkey array includes both Cmd+Shift+C (macOS) and Ctrl+Shift+C (Win/Linux).
	allKeys := []string{"c", "cmd", "shift", "c", "ctrl", "shift"}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Block until the hotkey is pressed.
				// The listener is configured for Cmd + Shift + C OR Ctrl + Shift + C.
				// addEvent returns true when the key combination is detected.

				if addEvent(allKeys...) {
					// Read clipboard and invoke callback
					item, err := ReadClipboard()

					// add hotkey event error handling
					if err != nil {
						fmt.Printf("Warning: Failed to read clipboard after hotkey: %v\n", err)
					} else if item != nil {
						cb(item)
					}
					// Small debounce to avoid multiple rapid triggers
					time.Sleep(200 * time.Millisecond)
				} else {
					// If AddEvent returned false immediately, yield to avoid busy loop
					time.Sleep(50 * time.Millisecond)
				}
			}
		}
	}()

	return nil
}

// GetClipboardItem is a Wails-exposed function to manually get clipboard content
func (a *App) GetClipboardItem() (*ClipboardItem, error) {
	return ReadClipboard()
}
