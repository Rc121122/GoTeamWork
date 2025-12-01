package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/go-vgo/robotgo"
	hook "github.com/robotn/gohook"
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

const (
	clipboardShareCooldown = 250 * time.Millisecond
	clipboardCacheTTL      = 8 * time.Second
)

var (
	addEvent         func(keys ...string) bool
	getMousePosition func() (int, int)

	clipboardHotkeyCombos = [][]string{
		{"c", "cmd"},
		{"c", "ctrl"},
	}
)

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

// StartClipboardMonitor starts monitoring for the copy hotkey (Cmd/Ctrl + C)
func (a *App) StartClipboardMonitor() {
	a.clipboardMonitorOnce.Do(func() {
		if a.ctx == nil {
			fmt.Println("Clipboard monitor skipped: no app context")
			return
		}

		// Check clipboard permissions for all platforms
		if !a.ensureAccessibilityPermission() {
			fmt.Println("Clipboard permission denied; clipboard monitor disabled")
			return
		}

		addEvent = func(keys ...string) bool {
			if len(keys) == 0 {
				return false
			}
			return hook.AddEvents(keys[0], keys[1:]...)
		}
		getMousePosition = robotgo.GetMousePos

		ctx, cancel := context.WithCancel(a.ctx)
		a.clipboardHotkeyCancel = cancel

		if err := StartClipboardHotkey(ctx, func(item *ClipboardItem, screenX, screenY int) {
			a.prepareClipboardShare(item, screenX, screenY)
		}); err != nil {
			fmt.Printf("StartClipboardHotkey failed: %v\n", err)
		}
	})
}

// handleClipboardCopy processes a copied clipboard item
func (a *App) handleClipboardCopy(item *ClipboardItem) {
	if item == nil {
		return
	}

	fmt.Printf("Clipboard copied: type=%s\n", item.Type)

	if item.Type == ClipboardText {
		item.Text = sanitizeClipboardText(item.Text)
		if item.Text == "" {
			fmt.Println("Clipboard text empty after sanitization; skipping broadcast")
			return
		}
	}

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
	a.historyPool.AddOperation(roomID, OpAdd, itemID, histItem)

	// Broadcast sanitized clipboard snapshot to all connected users
	a.sseManager.BroadcastToAll(EventClipboardCopied, item)
}

// StartClipboardHotkey listens for Cmd+C (macOS) and Ctrl+C (Win/Linux) global hotkeys.
// When the hotkey is detected, it reads the system clipboard and invokes cb with the
// ClipboardItem and the current mouse position.
func StartClipboardHotkey(ctx context.Context, cb func(*ClipboardItem, int, int)) error {
	if cb == nil {
		return errors.New("clipboard hotkey callback is required")
	}
	if addEvent == nil {
		return errors.New("global hotkey detector is not configured")
	}
	if getMousePosition == nil {
		return errors.New("mouse position provider is not configured")
	}

	if err := clipboard.Init(); err != nil {
		return fmt.Errorf("failed to init clipboard: %w", err)
	}

	for _, combo := range clipboardHotkeyCombos {
		keys := cloneKeys(combo)
		go monitorHotkey(ctx, keys, cb)
	}

	return nil
}

func monitorHotkey(ctx context.Context, combo []string, cb func(*ClipboardItem, int, int)) {
	for {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
		}

		if addEvent(combo...) {
			item, err := ReadClipboard()
			if err != nil {
				fmt.Printf("Warning: failed to read clipboard after hotkey: %v\n", err)
				time.Sleep(clipboardShareCooldown)
				continue
			}

			x, y := getMousePosition()
			cb(item, x, y)
			time.Sleep(clipboardShareCooldown)
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func cloneKeys(keys []string) []string {
	dup := make([]string, len(keys))
	copy(dup, keys)
	return dup
}

func (a *App) ensureAccessibilityPermission() bool {
	if runtime.GOOS == "darwin" {
		if hasAccessibilityPermission() {
			a.emitClipboardPermissionEvent(true, "")
			return true
		}

		a.emitClipboardPermissionEvent(false, "GOproject needs Accessibility access to watch Cmd+C events.")
		if !requestAccessibilityPermission() {
			return false
		}

		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if hasAccessibilityPermission() {
				a.emitClipboardPermissionEvent(true, "")
				return true
			}
			time.Sleep(500 * time.Millisecond)
		}

		granted := hasAccessibilityPermission()
		if granted {
			a.emitClipboardPermissionEvent(true, "")
		}
		return granted
	} else {
		// For Windows and Linux, check clipboard access
		if hasAccessibilityPermission() {
			a.emitClipboardPermissionEvent(true, "")
			return true
		}

		// Try to request permission
		a.emitClipboardPermissionEvent(false, "GOproject needs clipboard access. Please ensure no other application is locking the clipboard.")
		if !requestAccessibilityPermission() {
			return false
		}

		// Give a moment for permission to be granted
		time.Sleep(1 * time.Second)

		granted := hasAccessibilityPermission()
		if granted {
			a.emitClipboardPermissionEvent(true, "")
		}
		return granted
	}
}

func (a *App) prepareClipboardShare(item *ClipboardItem, screenX, screenY int) {
	a.cacheClipboardItem(item)
	a.emitClipboardButtonEvent(screenX, screenY)
}

func (a *App) cacheClipboardItem(item *ClipboardItem) {
	a.pendingClipboardMu.Lock()
	defer a.pendingClipboardMu.Unlock()
	a.pendingClipboardItem = item
	a.pendingClipboardAt = time.Now()
}

func (a *App) consumePendingClipboardItem() *ClipboardItem {
	a.pendingClipboardMu.Lock()
	defer a.pendingClipboardMu.Unlock()

	if a.pendingClipboardItem == nil {
		return nil
	}
	if time.Since(a.pendingClipboardAt) > clipboardCacheTTL {
		a.pendingClipboardItem = nil
		return nil
	}

	item := a.pendingClipboardItem
	a.pendingClipboardItem = nil
	return item
}

// ShareSystemClipboard publishes the most recent clipboard capture.
// If the cached value expired, it re-reads the live clipboard as a fallback.
func (a *App) ShareSystemClipboard() (bool, error) {
	item := a.consumePendingClipboardItem()
	if item == nil {
		var err error
		item, err = ReadClipboard()
		if err != nil {
			return false, err
		}
	}

	a.handleClipboardCopy(item)
	return true, nil
}

// GetClipboardItem is a Wails-exposed function to manually get clipboard content
func (a *App) GetClipboardItem() (*ClipboardItem, error) {
	return ReadClipboard()
}
