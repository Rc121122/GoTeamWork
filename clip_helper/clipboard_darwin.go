//go:build darwin
// +build darwin

package clip_helper

import (
	"fmt"

	"golang.design/x/clipboard"
)

// ReadClipboard reads the current clipboard content and returns a ClipboardItem
func ReadClipboard() (*ClipboardItem, error) {
	// Initialize clipboard if needed
	err := clipboard.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize clipboard: %w", err)
	}

	// Check for files first (MacOS specific)
	// We can't easily "peek" without reading, but getFilePathsFromPasteboard is cheap?
	// Actually, usually we check text/image.
	// But if we copied files, clipboard.Read(clipboard.FmtText) might return the file names or empty.
	// Let's check files first.
	if filePaths := getFilePathsFromPasteboard(); len(filePaths) > 0 {
		// Return immediately with file paths, zip in background if needed
		// For now, we return the item with Files populated and empty ZipData.
		// The caller (App.handleClipboardCopy) should handle the zipping asynchronously.
		return &ClipboardItem{
			Type:  ClipboardFile,
			Files: filePaths,
			Text:  fmt.Sprintf("%d files selected", len(filePaths)),
		}, nil
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
