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

type fileShareKind string

const (
	kindSingle fileShareKind = "single_file"
	kindFiles  fileShareKind = "multiple_files"
	kindDirs   fileShareKind = "folders"
	kindMixed  fileShareKind = "mixed"
)

var (
	getMousePosition func() (int, int)
)

// StartClipboardMonitor starts monitoring for clipboard changes
func (a *App) StartClipboardMonitor() {
	fmt.Println("[DEBUG] Starting clipboard monitor")
	a.clipboardMonitorOnce.Do(func() {
		if a.ctx == nil {
			fmt.Println("[DEBUG] Clipboard monitor skipped: no app context")
			return
		}

		// Check clipboard permissions for all platforms
		if !a.ensureAccessibilityPermission() {
			fmt.Println("[DEBUG] Clipboard permission denied; clipboard monitor disabled")
			// return // Don't return, try to proceed as clipboard read might still work
		}

		getMousePosition = robotgo.GetMousePos

		ctx, cancel := context.WithCancel(a.ctx)
		a.clipboardHotkeyCancel = cancel

		if err := StartClipboardWatcher(ctx, func(item *clip_helper.ClipboardItem, screenX, screenY int) {
			a.prepareClipboardShare(item, screenX, screenY)
		}); err != nil {
			fmt.Printf("[DEBUG] StartClipboardWatcher failed: %v\n", err)
		}
	})
}

func classifyClipboardPaths(paths []string) (fileShareKind, error) {
	if len(paths) == 0 {
		return kindMixed, fmt.Errorf("no paths to classify")
	}

	files := 0
	dirs := 0

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return kindMixed, fmt.Errorf("stat failed for %s: %w", p, err)
		}
		if info.IsDir() {
			dirs++
		} else {
			files++
		}
	}

	if files == 1 && dirs == 0 {
		return kindSingle, nil
	}
	if dirs > 0 && files == 0 {
		return kindDirs, nil
	}
	if dirs > 0 && files > 0 {
		return kindMixed, nil
	}
	return kindFiles, nil
}

func totalClipboardSize(paths []string) (int64, error) {
	var total int64
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return 0, fmt.Errorf("stat failed for %s: %w", p, err)
		}
		if info.IsDir() {
			err = filepath.Walk(p, func(path string, fi os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if fi.Mode().IsRegular() {
					total += fi.Size()
				}
				return nil
			})
			if err != nil {
				return 0, err
			}
			continue
		}
		total += info.Size()
	}
	return total, nil
}

// reportTransferTiming calculates and displays transfer timing and speed
func (a *App) reportTransferTiming(operation string) {
	if a.shareStartTime.IsZero() || a.totalBytesTransferred == 0 {
		return
	}

	elapsed := time.Since(a.shareStartTime)
	seconds := elapsed.Seconds()
	gbPerSecond := float64(a.totalBytesTransferred) / (1024 * 1024 * 1024) / seconds

	fmt.Printf("[TIMER] %s completed in %.2fs - %.2f GB/s (%s transferred)\n",
		operation, seconds, gbPerSecond, clip_helper.HumanFileSize(a.totalBytesTransferred))

	// Reset for next measurement
	a.shareStartTime = time.Time{}
	a.totalBytesTransferred = 0
}

// handleClipboardCopy processes a copied clipboard item
func (a *App) handleClipboardCopy(item *clip_helper.ClipboardItem) {
	fmt.Printf("[DEBUG] handleClipboardCopy: processing item type=%s\n", item.Type)
	if item == nil {
		fmt.Println("[DEBUG] handleClipboardCopy: item is nil")
		return
	}

	if item.Type == clip_helper.ClipboardText {
		fmt.Printf("[DEBUG] Processing text clipboard: %s\n", item.Text)
		item.Text = sanitizeClipboardText(item.Text)
		if item.Text == "" {
			fmt.Println("[DEBUG] Clipboard text empty after sanitization; skipping broadcast")
			return
		}
	}

	// Check file sizes for file clipboard items
	if item.Type == clip_helper.ClipboardFile && len(item.Files) > 0 {
		fmt.Printf("[DEBUG] Checking %d files...\n", len(item.Files))
		const maxIndividualFileSize = 100 * 1024 * 1024 * 1024 // 100GB per file
		const maxTotalFiles = 100                               // Maximum 100 files

		if len(item.Files) > maxTotalFiles {
			fmt.Printf("[DEBUG] Too many files: %d (max %d), skipping\n", len(item.Files), maxTotalFiles)
			return
		}

		for _, filePath := range item.Files {
			info, err := os.Stat(filePath)
			if err != nil {
				fmt.Printf("[DEBUG] Failed to stat file %s: %v, skipping\n", filePath, err)
				return
			}
			if info.Size() > maxIndividualFileSize {
				fmt.Printf("[DEBUG] File %s too large: %d bytes (max %d bytes), skipping\n",
					filePath, info.Size(), maxIndividualFileSize)
				return
			}
		}
		fmt.Printf("[DEBUG] File validation passed: %d files\n", len(item.Files))
	}

	// Assume roomID from current room or something, but since broadcast to all, perhaps global or per room.
	// For now, use a default room or broadcast to all rooms.
	// To fit, perhaps add to a global room or modify.

	if a.Mode == "client" {
		fmt.Println("[DEBUG] Client mode: uploading clipboard item")
		// Upload to server
		op, err := a.networkClient.UploadClipboardItem(item, a.currentUser.ID, a.currentUser.Name)
		if err != nil {
			fmt.Printf("[DEBUG] Failed to upload clipboard item: %v\n", err)
			return
		}
		fmt.Printf("[DEBUG] Upload successful, op ID: %s\n", op.ID)

		// If file, start async archive processing
		if item.Type == clip_helper.ClipboardFile && item.ArchiveFilePath == "" && len(item.Files) > 0 {
			fmt.Println("[DEBUG] Starting async archive/process for client")
			go a.processFileArchive("global", op.ItemID, item, op.ID)
		}
		return
	}

	// Host logic: require room
	if a.currentUser.RoomID == nil {
		fmt.Println("[DEBUG] Host mode: not in a room, cannot share")
		// Host not in a room, cannot share
		return
	}
	roomID := *a.currentUser.RoomID
	fmt.Printf("[DEBUG] Host mode: sharing to room %s\n", roomID)

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
		fmt.Printf("[DEBUG] Broadcasting clipboard copy to %d members\n", len(members))
		a.sseManager.BroadcastToUsers(members, EventClipboardCopied, op, "")
	}

	// If it's a file type and no archive exists, start async archiving
	if item.Type == clip_helper.ClipboardFile && item.ArchiveFilePath == "" && len(item.Files) > 0 {
		fmt.Println("[DEBUG] Starting async archive process for host")
		go a.processFileArchive(roomID, itemID, item, "")
	}
}
func (a *App) processFileArchive(roomID, itemID string, item *clip_helper.ClipboardItem, serverOpID string) {
	kind, err := classifyClipboardPaths(item.Files)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to classify files: %v\n", err)
		return
	}

	fmt.Printf("[DEBUG] Starting async archive for item %s (%s), %d entries\n", itemID, kind, len(item.Files))

	// Direct send for single-file shares
	if kind == kindSingle {
		fmt.Printf("[DEBUG] Single file detected, processing directly\n")
		a.processSingleFileShare(roomID, itemID, item, serverOpID)
		return
	}

	// Check total file size before archiving (limit to 100GB)
	const maxArchiveSize = 100 * 1024 * 1024 * 1024 // 100GB
	totalSize, err := totalClipboardSize(item.Files)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to calculate total size: %v\n", err)
		return
	}

	fmt.Printf("[DEBUG] Total file size: %d bytes, proceeding with tar archiving\n", totalSize)

	// Stream directly to final tar file (no temp file overhead)
	archiveFilePath := filepath.Join(a.tempDir, itemID+".tar")
	archiveFile, err := os.Create(archiveFilePath)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to create tar file: %v\n", err)
		return
	}
	defer archiveFile.Close()

	fmt.Printf("[DEBUG] Created tar file: %s\n", archiveFilePath)

	// Set up progress reporting for large archives (>100MB)
	var progressCallback clip_helper.ProgressCallback
	if totalSize > 100*1024*1024 {
		progressCallback = func(processedBytes int64, totalBytes int64, currentFile string) {
			fmt.Printf("[DEBUG] Archiving progress: %s (%d MB processed)\n", currentFile, processedBytes/(1024*1024))
		}
	}

	// Choose tar implementation based on size and settings
	var archiveErr error
	if a.useFastTar && totalSize > 1024*1024*1024 { // Use fast tar for files >1GB
		fmt.Printf("[DEBUG] Using fast tar library for large archive\n")
		archiveErr = clip_helper.TarPathsFast(item.Files, archiveFile)
	} else {
		fmt.Printf("[DEBUG] Using optimized standard tar library\n")
		archiveErr = clip_helper.TarPathsWithProgress(item.Files, archiveFile, progressCallback)
	}

	if archiveErr != nil {
		fmt.Printf("[DEBUG] Failed to create tar archive: %v\n", archiveErr)
		archiveFile.Close()
		os.Remove(archiveFilePath)
		return
	}

	// Close the file to ensure all data is written
	if err := archiveFile.Close(); err != nil {
		fmt.Printf("[DEBUG] Failed to close tar file: %v\n", err)
		os.Remove(archiveFilePath)
		return
	}

	// Get final archive size
	archiveInfo, err := os.Stat(archiveFilePath)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to stat tar file: %v\n", err)
		os.Remove(archiveFilePath)
		return
	}

	archiveSize := archiveInfo.Size()

	// Update the item in history pool
	a.mu.Lock()
	item.ArchiveFilePath = archiveFilePath
	item.Text = fmt.Sprintf("%d items archived (ready)", len(item.Files))
	a.mu.Unlock()

	fmt.Printf("[DEBUG] Direct tar archive completed for item %s, size: %d bytes\n", itemID, archiveSize)

	if a.Mode == "client" {
		if serverOpID != "" {
			// Upload archive file from disk instead of loading into memory
			fmt.Printf("[DEBUG] Uploading archive file to server op %s\n", serverOpID)
			if err := a.networkClient.UploadZipFile(serverOpID, archiveFilePath); err != nil {
				fmt.Printf("[DEBUG] Failed to upload archive file: %v\n", err)
				return
			}
			fmt.Printf("[DEBUG] Uploaded archive file for op %s\n", serverOpID)
			a.reportTransferTiming("Archive upload")
		}
		return
	}

	fmt.Printf("[DEBUG] Broadcasting archive update for item %s\n", itemID)
	a.broadcastClipboardUpdate(roomID, itemID)
}

func (a *App) processSingleFileShare(roomID, itemID string, item *clip_helper.ClipboardItem, serverOpID string) {
	fmt.Printf("[DEBUG] Processing single file share for item %s\n", itemID)
	filePath := item.Files[0]
	info, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to stat single file %s: %v\n", filePath, err)
		return
	}

	fileName := filepath.Base(filePath)
	mimeType := detectMimeType(fileName)
	thumb := buildFileThumb(fileName)

	fmt.Printf("[DEBUG] Single file: %s (%s, %d bytes)\n", fileName, mimeType, info.Size())

	// Save the single file to temp directory without loading into memory
	singleFilePath := filepath.Join(a.tempDir, itemID+"_"+fileName)
	fmt.Printf("[DEBUG] Copying single file to temp: %s -> %s\n", filePath, singleFilePath)
	src, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to open single file %s: %v\n", filePath, err)
		return
	}
	defer src.Close()

	dest, err := os.Create(singleFilePath)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to create temp single file: %v\n", err)
		return
	}
	if _, err := io.Copy(dest, src); err != nil {
		dest.Close()
		fmt.Printf("[DEBUG] Failed to copy single file: %v\n", err)
		return
	}
	if err := dest.Close(); err != nil {
		fmt.Printf("[DEBUG] Failed to finalize single file copy: %v\n", err)
		return
	}
	fmt.Printf("[DEBUG] Single file copied to temp successfully\n")

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

	fmt.Printf("[DEBUG] Prepared single file share %s (%d bytes)\n", fileName, info.Size())

	if a.Mode == "client" {
		if serverOpID != "" {
			fmt.Printf("[DEBUG] Client mode: uploading single file for op %s\n", serverOpID)
			// Upload single file from disk instead of loading into memory
			if err := a.networkClient.UploadSingleFile(serverOpID, singleFilePath, fileName, mimeType, info.Size(), thumb); err != nil {
				fmt.Printf("[DEBUG] Failed to upload single file: %v\n", err)
			}
			fmt.Printf("[DEBUG] Uploaded single file for op %s\n", serverOpID)
			a.reportTransferTiming("Single file upload")
		}
		return
	}

	fmt.Printf("[DEBUG] Host mode: broadcasting single file update for room %s\n", roomID)
	a.broadcastClipboardUpdate(roomID, itemID)
}

func (a *App) broadcastClipboardUpdate(roomID, itemID string) {
	fmt.Printf("[DEBUG] Broadcasting clipboard update for item %s in room %s\n", itemID, roomID)
	ops := a.historyPool.GetOperations(roomID, "", "")
	var targetOp *Operation
	for _, op := range ops {
		if op.ItemID == itemID {
			targetOp = op
			break
		}
	}

	if targetOp == nil {
		fmt.Printf("[DEBUG] Warning: Could not find operation for item %s to broadcast update\n", itemID)
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
		fmt.Printf("[DEBUG] Broadcasting clipboard update for item %s to %d members\n", itemID, len(members))
		a.sseManager.BroadcastToUsers(members, EventClipboardUpdated, targetOp, "")
		fmt.Printf("[DEBUG] Clipboard update broadcast completed for item %s\n", itemID)
	} else {
		fmt.Printf("[DEBUG] Room %s not found, cannot broadcast clipboard update\n", roomID)
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
	a.shareStartTime = time.Now() // Record start time for performance measurement

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

	// Calculate total bytes for performance measurement
	if item.Type == clip_helper.ClipboardFile && len(item.Files) > 0 {
		totalBytes, err := totalClipboardSize(item.Files)
		if err == nil {
			a.totalBytesTransferred = totalBytes
			fmt.Printf("[TIMER] Starting transfer of %d bytes (%s)\n", totalBytes, clip_helper.HumanFileSize(totalBytes))
		}
	}

	fmt.Printf("Processing clipboard item: Type=%s, Files=%d\n", item.Type, len(item.Files))
	a.handleClipboardCopy(item)
	return true, nil
}

// GetClipboardItem is a Wails-exposed function to manually get clipboard content
func (a *App) GetClipboardItem() (*clip_helper.ClipboardItem, error) {
	return clip_helper.ReadClipboard()
}
