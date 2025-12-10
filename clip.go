package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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

	if a.Mode == "client" {
		// Upload to server
		op, err := a.networkClient.UploadClipboardItem(item)
		if err != nil {
			fmt.Printf("Failed to upload clipboard item: %v\n", err)
			return
		}

		// If file, start async zip
		if item.Type == clip_helper.ClipboardFile && len(item.ZipData) == 0 && len(item.Files) > 0 {
			go a.processFileZip("global", op.ItemID, item, op.ID)
		}
		return
	}

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

	// Broadcast sanitized clipboard snapshot to all connected users
	a.sseManager.BroadcastToAll(EventClipboardCopied, op)

	// If it's a file type and ZipData is empty, start async zipping
	if item.Type == clip_helper.ClipboardFile && len(item.ZipData) == 0 && len(item.Files) > 0 {
		go a.processFileZip(roomID, itemID, item, "")
	}
}

func (a *App) processFileZip(roomID, itemID string, item *clip_helper.ClipboardItem, serverOpID string) {
	fmt.Printf("Starting async zip for item %s with %d files\n", itemID, len(item.Files))
	
	// Create a temp zip file
	tmpFile, err := os.CreateTemp("", "clipboard_files_*.zip")
	if err != nil {
		fmt.Printf("Failed to create temp zip file: %v\n", err)
		return
	}
	defer os.Remove(tmpFile.Name()) // Clean up temp file after reading
	defer tmpFile.Close()

	if err := clip_helper.ZipFiles(item.Files, tmpFile); err != nil {
		fmt.Printf("Failed to zip files: %v\n", err)
		return
	}

	// Read the zip file back into memory
	if _, err := tmpFile.Seek(0, 0); err != nil {
		fmt.Printf("Failed to seek temp zip file: %v\n", err)
		return
	}

	zipData, err := io.ReadAll(tmpFile)
	if err != nil {
		fmt.Printf("Failed to read temp zip file: %v\n", err)
		return
	}

	// Update the item in history pool
	a.mu.Lock()
	// We need to find the operation and update the item
	// This is a bit tricky since HistoryPool manages operations.
	// Let's add a method to HistoryPool to update item data?
	// Or just update the item pointer since we passed it?
	// Yes, item is a pointer, so updating it here updates it in the history pool if it's the same instance.
	item.ZipData = zipData
	item.Text = fmt.Sprintf("%d files compressed (ready)", len(item.Files))
	a.mu.Unlock()

	fmt.Printf("Async zip completed for item %s, size: %d bytes\n", itemID, len(zipData))
	
	if a.Mode == "client" {
		if serverOpID != "" {
			if err := a.networkClient.UploadZipData(serverOpID, zipData); err != nil {
				fmt.Printf("Failed to upload zip data: %v\n", err)
			} else {
				fmt.Printf("Uploaded zip data for op %s\n", serverOpID)
			}
		}
		return
	}

	// Broadcast update
	ops := a.historyPool.GetOperations(roomID, "")
	var targetOp *Operation
	for _, op := range ops {
		if op.ItemID == itemID {
			targetOp = op
			break
		}
	}

	if targetOp != nil {
		a.sseManager.BroadcastToAll(EventClipboardUpdated, targetOp)
	}
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

