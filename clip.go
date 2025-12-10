package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"GOproject/clip_helper"

	"github.com/go-vgo/robotgo"
	"golang.design/x/clipboard"
)

var (
	getMousePosition func() (int, int)
)

// StartClipboardMonitor starts monitoring for clipboard changes
func (a *App) StartClipboardMonitor() {
	a.clipboardMonitorOnce.Do(func() {
		if a.ctx == nil {
			fmt.Println("Clipboard monitor skipped: no app context")
			return
		}

		// Check clipboard permissions for all platforms
		if !a.ensureAccessibilityPermission() {
			fmt.Println("Clipboard permission denied; clipboard monitor disabled")
			// return // Don't return, try to proceed as clipboard read might still work
		}

		getMousePosition = robotgo.GetMousePos

		ctx, cancel := context.WithCancel(a.ctx)
		a.clipboardHotkeyCancel = cancel

		if err := StartClipboardWatcher(ctx, func(item *clip_helper.ClipboardItem, screenX, screenY int) {
			a.prepareClipboardShare(item, screenX, screenY)
		}); err != nil {
			fmt.Printf("StartClipboardWatcher failed: %v\n", err)
		}
	})
}

// handleClipboardCopy processes a copied clipboard item
func (a *App) handleClipboardCopy(item *clip_helper.ClipboardItem) {
	if item == nil {
		return
	}

	fmt.Printf("Clipboard copied: type=%s\n", item.Type)

	if item.Type == clip_helper.ClipboardText {
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

// StartClipboardWatcher listens for clipboard changes.
// When a change is detected, it reads the system clipboard and invokes cb with the
// ClipboardItem and the current mouse position.
func StartClipboardWatcher(ctx context.Context, cb func(*clip_helper.ClipboardItem, int, int)) error {
	if cb == nil {
		return errors.New("clipboard callback is required")
	}
	if getMousePosition == nil {
		return errors.New("mouse position provider is not configured")
	}

	if err := clipboard.Init(); err != nil {
		return fmt.Errorf("failed to init clipboard: %w", err)
	}

	// Watch for text changes
	chText := clipboard.Watch(ctx, clipboard.FmtText)
	// Watch for image changes
	chImage := clipboard.Watch(ctx, clipboard.FmtImage)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-chText:
				handleClipboardChange(cb)
			case <-chImage:
				handleClipboardChange(cb)
			}
		}
	}()

	return nil
}

func handleClipboardChange(cb func(*clip_helper.ClipboardItem, int, int)) {
	// Give a small delay to ensure clipboard is ready
	time.Sleep(100 * time.Millisecond)

	item, err := clip_helper.ReadClipboard()
	if err != nil {
		fmt.Printf("Warning: failed to read clipboard after change: %v\n", err)
		return
	}

	x, y := getMousePosition()
	cb(item, x, y)
}

func (a *App) ensureAccessibilityPermission() bool {
	if clip_helper.HasAccessibilityPermission() {
		a.emitClipboardPermissionEvent(true, "")
		return true
	}

	a.emitClipboardPermissionEvent(false, "GOproject needs Accessibility access to watch clipboard events.")
	if !clip_helper.RequestAccessibilityPermission() {
		return false
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if clip_helper.HasAccessibilityPermission() {
			a.emitClipboardPermissionEvent(true, "")
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}

	granted := clip_helper.HasAccessibilityPermission()
	if granted {
		a.emitClipboardPermissionEvent(true, "")
	}
	return granted
}

func (a *App) prepareClipboardShare(item *clip_helper.ClipboardItem, screenX, screenY int) {
	a.cacheClipboardItem(item)
	a.emitClipboardButtonEvent(screenX, screenY)
}

func (a *App) cacheClipboardItem(item *clip_helper.ClipboardItem) {
	a.pendingClipboardMu.Lock()
	defer a.pendingClipboardMu.Unlock()
	a.pendingClipboardItem = item
	a.pendingClipboardAt = time.Now()
}

func (a *App) consumePendingClipboardItem() *clip_helper.ClipboardItem {
	a.pendingClipboardMu.Lock()
	defer a.pendingClipboardMu.Unlock()

	if a.pendingClipboardItem == nil {
		return nil
	}
	if time.Since(a.pendingClipboardAt) > clip_helper.ClipboardCacheTTL {
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
		item, err = clip_helper.ReadClipboard()
		if err != nil {
			return false, err
		}
	}

	a.handleClipboardCopy(item)
	return true, nil
}

// GetClipboardItem is a Wails-exposed function to manually get clipboard content
func (a *App) GetClipboardItem() (*clip_helper.ClipboardItem, error) {
	return clip_helper.ReadClipboard()
}

