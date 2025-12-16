package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		fmt.Println("handleClipboardCopy: item is nil")
		return
	}

	fmt.Printf("handleClipboardCopy: processing item type=%s\n", item.Type)

	if item.Type == clip_helper.ClipboardText {
		item.Text = sanitizeClipboardText(item.Text)
		if item.Text == "" {
			fmt.Println("Clipboard text empty after sanitization; skipping broadcast")
			return
		}
	}

	// Check file sizes for file clipboard items
	if item.Type == clip_helper.ClipboardFile && len(item.Files) > 0 {
		fmt.Printf("Checking %d files...\n", len(item.Files))
		const maxIndividualFileSize = 10 * 1024 * 1024 * 1024 // 10GB per file
		const maxTotalFiles = 100                             // Maximum 100 files

		if len(item.Files) > maxTotalFiles {
			fmt.Printf("Too many files: %d (max %d), skipping\n", len(item.Files), maxTotalFiles)
			return
		}

		for _, filePath := range item.Files {
			info, err := os.Stat(filePath)
			if err != nil {
				fmt.Printf("Failed to stat file %s: %v, skipping\n", filePath, err)
				return
			}
			if info.Size() > maxIndividualFileSize {
				fmt.Printf("File %s too large: %d bytes (max %d bytes), skipping\n",
					filePath, info.Size(), maxIndividualFileSize)
				return
			}
		}
		fmt.Printf("File validation passed: %d files\n", len(item.Files))
	}

	// Assume roomID from current room or something, but since broadcast to all, perhaps global or per room.
	// For now, use a default room or broadcast to all rooms.
	// To fit, perhaps add to a global room or modify.

	if a.Mode == "client" {
		fmt.Println("Client mode: uploading clipboard item")
		// Upload to server
		op, err := a.networkClient.UploadClipboardItem(item, a.currentUser.ID, a.currentUser.Name)
		if err != nil {
			fmt.Printf("Failed to upload clipboard item: %v\n", err)
			return
		}
		fmt.Printf("Upload successful, op ID: %s\n", op.ID)

		// If file, start async zip
		if item.Type == clip_helper.ClipboardFile && item.ZipFilePath == "" && len(item.Files) > 0 {
			fmt.Println("Starting async zip/process for client")
			go a.processFileZip("global", op.ItemID, item, op.ID)
		}
		return
	}

	// Host logic: require room
	if a.currentUser.RoomID == nil {
		fmt.Println("Host mode: not in a room, cannot share")
		// Host not in a room, cannot share
		return
	}
	roomID := *a.currentUser.RoomID
	fmt.Printf("Host mode: sharing to room %s\n", roomID)

	// Create item ID
	itemID := fmt.Sprintf("clip_%d", time.Now().UnixNano())

	// Create Item
	histItem := &Item{
		ID:   itemID,
		Type: ItemClipboard,
		Data: item,
	}

	// Add operation
	op := a.historyPool.AddOperation(roomID, OpAdd, itemID, histItem, a.currentUser.ID, a.currentUser.Name)

	// Broadcast to room members
	a.mu.RLock()
	room, roomExists := a.rooms[roomID]
	var members []string
	if roomExists {
		members = append(members, room.UserIDs...)
	}
	a.mu.RUnlock()

	if roomExists {
		a.sseManager.BroadcastToUsers(members, EventClipboardCopied, op, "")
	}

	// If it's a file type and ZipData is empty, start async zipping
	if item.Type == clip_helper.ClipboardFile && item.ZipFilePath == "" && len(item.Files) > 0 {
		go a.processFileZip(roomID, itemID, item, "")
	}
}

func (a *App) processFileZip(roomID, itemID string, item *clip_helper.ClipboardItem, serverOpID string) {
	fmt.Printf("Starting async zip for item %s with %d files\n", itemID, len(item.Files))

	// Direct send for single-file shares
	if len(item.Files) == 1 {
		a.processSingleFileShare(roomID, itemID, item, serverOpID)
		return
	}

	// Check total file size before compression (limit to 10GB)
	const maxZipSize = 10 * 1024 * 1024 * 1024 // 10GB
	var totalSize int64 = 0

	for _, filePath := range item.Files {
		info, err := os.Stat(filePath)
		if err != nil {
			fmt.Printf("Failed to stat file %s: %v\n", filePath, err)
			continue
		}
		totalSize += info.Size()
	}

	if totalSize > maxZipSize {
		fmt.Printf("Total file size %d bytes exceeds limit of %d bytes, skipping compression\n", totalSize, maxZipSize)
		a.mu.Lock()
		item.Text = fmt.Sprintf("Files too large to compress (%d MB > %d MB limit)", totalSize/(1024*1024), maxZipSize/(1024*1024))
		a.mu.Unlock()
		return
	}

	fmt.Printf("Total file size: %d bytes, proceeding with compression\n", totalSize)

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

	// Check the final zip file size
	zipInfo, err := tmpFile.Stat()
	if err != nil {
		fmt.Printf("Failed to stat zip file: %v\n", err)
		return
	}

	zipSize := zipInfo.Size()
	if zipSize > maxZipSize {
		fmt.Printf("Compressed zip size %d bytes exceeds limit of %d bytes\n", zipSize, maxZipSize)
		a.mu.Lock()
		item.Text = fmt.Sprintf("Compressed size too large (%d MB > %d MB limit)", zipSize/(1024*1024), maxZipSize/(1024*1024))
		a.mu.Unlock()
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

	// Save the zip file to temp directory
	zipFilePath := filepath.Join(a.tempDir, itemID+".zip")
	if err := os.WriteFile(zipFilePath, zipData, 0644); err != nil {
		fmt.Printf("Failed to write zip file: %v\n", err)
		return
	}

	// Update the item in history pool
	a.mu.Lock()
	item.ZipFilePath = zipFilePath
	item.Text = fmt.Sprintf("%d files compressed (ready)", len(item.Files))
	a.mu.Unlock()

	fmt.Printf("Async zip completed for item %s, size: %d bytes\n", itemID, len(zipData))

	if a.Mode == "client" {
		if serverOpID != "" {
			// Read zip data from file for upload
			uploadedData, err := os.ReadFile(zipFilePath)
			if err != nil {
				fmt.Printf("Failed to read zip file for upload: %v\n", err)
				return
			}
			if err := a.networkClient.UploadZipData(serverOpID, uploadedData); err != nil {
				fmt.Printf("Failed to upload zip data: %v\n", err)
				return
			}
			fmt.Printf("Uploaded zip data for op %s\n", serverOpID)
		}
		return
	}

	a.broadcastClipboardUpdate(roomID, itemID)
}

func (a *App) processSingleFileShare(roomID, itemID string, item *clip_helper.ClipboardItem, serverOpID string) {
	filePath := item.Files[0]
	info, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Failed to stat single file %s: %v\n", filePath, err)
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Failed to read single file %s: %v\n", filePath, err)
		return
	}

	fileName := filepath.Base(filePath)
	mimeType := detectMimeType(fileName)
	thumb := buildFileThumb(fileName)

	// Save the single file to temp directory
	singleFilePath := filepath.Join(a.tempDir, itemID+"_"+fileName)
	if err := os.WriteFile(singleFilePath, data, 0644); err != nil {
		fmt.Printf("Failed to write single file: %v\n", err)
		return
	}

	a.mu.Lock()
	item.IsSingleFile = true
	item.SingleFileName = fileName
	item.SingleFileMime = mimeType
	item.SingleFileSize = info.Size()
	item.SingleFileThumb = thumb
	item.SingleFilePath = singleFilePath
	item.ZipData = nil
	item.Text = fmt.Sprintf("%s (%s) ready", fileName, clip_helper.HumanFileSize(info.Size()))
	a.mu.Unlock()

	fmt.Printf("Prepared single file share %s (%d bytes)\n", fileName, len(data))

	if a.Mode == "client" {
		if serverOpID != "" {
			// Read file data from disk for upload
			uploadedData, err := os.ReadFile(singleFilePath)
			if err != nil {
				fmt.Printf("Failed to read single file for upload: %v\n", err)
				return
			}
			if err := a.networkClient.UploadSingleFileData(serverOpID, uploadedData, fileName, mimeType, info.Size(), thumb); err != nil {
				fmt.Printf("Failed to upload single file data: %v\n", err)
			}
			fmt.Printf("Uploaded single file data for op %s\n", serverOpID)
		}
		return
	}

	a.broadcastClipboardUpdate(roomID, itemID)
}

func (a *App) broadcastClipboardUpdate(roomID, itemID string) {
	ops := a.historyPool.GetOperations(roomID, "", "")
	var targetOp *Operation
	for _, op := range ops {
		if op.ItemID == itemID {
			targetOp = op
			break
		}
	}

	if targetOp == nil {
		fmt.Printf("Warning: Could not find operation for item %s to broadcast update\n", itemID)
		return
	}

	a.mu.RLock()
	room, roomExists := a.rooms[roomID]
	var members []string
	if roomExists {
		members = append(members, room.UserIDs...)
	}
	a.mu.RUnlock()

	if roomExists {
		fmt.Printf("Broadcasting clipboard update for item %s\n", itemID)
		a.sseManager.BroadcastToUsers(members, EventClipboardUpdated, targetOp, "")
	}
}

func detectMimeType(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext != "" {
		if mt := mime.TypeByExtension(ext); mt != "" {
			return mt
		}
	}
	return "application/octet-stream"
}

func buildFileThumb(name string) string {
	ext := strings.TrimPrefix(strings.ToUpper(filepath.Ext(name)), ".")
	if ext == "" {
		ext = strings.ToUpper(name)
	}
	if len(ext) > 4 {
		ext = ext[:4]
	}
	return ext
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

	// On Windows, also poll for file changes since CF_HDROP is not watched by clipboard.Watch
	if runtime.GOOS == "windows" {
		go startWindowsFilePoller(ctx, cb)
	}

	return nil
}

// startWindowsFilePoller polls the clipboard for file changes on Windows
func startWindowsFilePoller(ctx context.Context, cb func(*clip_helper.ClipboardItem, int, int)) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastFilePathsHash string
	var lastDetectionTime time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			item, err := clip_helper.ReadClipboard()
			if err != nil {
				continue
			}

			// Only interested in file changes
			if item.Type != clip_helper.ClipboardFile || len(item.Files) == 0 {
				continue
			}

			// Create a hash of the current file paths to detect if they changed
			currentHash := fmt.Sprintf("%v", item.Files)

			// Only notify if the file set changed AND enough time has passed since last detection
			// This prevents multiple notifications for the same file set
			now := time.Now()
			if currentHash != lastFilePathsHash && now.Sub(lastDetectionTime) > 1*time.Second {
				lastDetectionTime = now
				lastFilePathsHash = currentHash
				x, y := getMousePosition()
				// Emit HUD first (nil item triggers HUD display)
				cb(nil, x, y)
				// Then provide the item
				cb(item, x, y)
			}
		}
	}
}

func handleClipboardChange(cb func(*clip_helper.ClipboardItem, int, int)) {
	x, y := getMousePosition()

	// Emit HUD immediately for faster response
	// Note: This assumes cb is a.prepareClipboardShare which emits the event
	cb(nil, x, y)

	// Then asynchronously read clipboard after delay
	go func() {
		time.Sleep(50 * time.Millisecond)

		item, err := clip_helper.ReadClipboard()
		if err != nil {
			fmt.Printf("Warning: failed to read clipboard after change: %v\n", err)
			return
		}

		// Update cache with actual item
		cb(item, x, y)
	}()
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
	if item == nil {
		// Emit HUD immediately for faster response
		a.emitClipboardButtonEvent(screenX, screenY)
	} else {
		// Cache the actual clipboard item
		a.cacheClipboardItem(item)
	}
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
	fmt.Println("ShareSystemClipboard called")
	item := a.consumePendingClipboardItem()
	if item == nil {
		fmt.Println("No pending item or expired, reading from clipboard directly")
		var err error
		item, err = clip_helper.ReadClipboard()
		if err != nil {
			fmt.Printf("Failed to read clipboard: %v\n", err)
			return false, err
		}
	} else {
		fmt.Println("Using pending clipboard item")
	}

	fmt.Printf("Processing clipboard item: Type=%s, Files=%d\n", item.Type, len(item.Files))
	a.handleClipboardCopy(item)
	return true, nil
}

// GetClipboardItem is a Wails-exposed function to manually get clipboard content
func (a *App) GetClipboardItem() (*clip_helper.ClipboardItem, error) {
	return clip_helper.ReadClipboard()
}
