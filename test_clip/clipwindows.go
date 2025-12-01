//go:build windows
// +build windows

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.design/x/clipboard"
)

func main() {
	fmt.Println("Welcome to clipboard testing for Windows, waiting for response, press \"Ctrl+C\" to copy here.")

	ctx := context.Background()

	// Watch for text changes
	chText := clipboard.Watch(ctx, clipboard.FmtText)

	// Watch for image changes (if supported)
	chImage := clipboard.Watch(ctx, clipboard.FmtImage)

	// Test initial clipboard content
	if err := clipboard.Init(); err != nil {
		fmt.Printf("Failed to initialize clipboard: %v\n", err)
		return
	}

	fmt.Println("Clipboard initialized successfully")

	// Read initial content
	if textData := clipboard.Read(clipboard.FmtText); len(textData) > 0 {
		fmt.Printf("Initial clipboard text: %s\n", string(textData))
	}

	if imgData := clipboard.Read(clipboard.FmtImage); len(imgData) > 0 {
		fmt.Printf("Initial clipboard has image data (%d bytes)\n", len(imgData))
	}

	fmt.Println("Monitoring clipboard changes...")

	for {
		select {
		case data := <-chText:
			text := string(data)

			// Skip empty or whitespace-only text
			if strings.TrimSpace(text) == "" {
				continue
			}

			fmt.Printf("You copied text: %s\n", text)

			// Check if it looks like file paths
			if isFilePath(text) {
				fmt.Printf("Detected file path(s): %s\n", text)
			} else if len(strings.TrimSpace(text)) > 0 {
				// Additional check: if text contains file-like patterns but not detected as full paths
				if strings.Contains(text, "\\") || strings.Contains(text, "/") {
					fmt.Printf("Possible file reference: %s\n", text)
				}
			}

			// Only modify clipboard if it's not already modified text (avoid infinite loop)
			if !strings.HasPrefix(text, "Modified: ") {
				// Test writing back to clipboard
				testText := fmt.Sprintf("Modified: %s", text)
				clipboard.Write(clipboard.FmtText, []byte(testText))
				fmt.Printf("Modified clipboard to: %s\n", testText)
			}

		case data := <-chImage:
			if len(data) > 0 {
				fmt.Printf("You copied an image (%d bytes)\n", len(data))

				// Test writing back (though we can't modify image data easily)
				fmt.Println("Image copied to clipboard")
			}

		case <-time.After(30 * time.Second):
			fmt.Println("No clipboard activity for 30 seconds...")
		}
	}
}

// isFilePath checks if the text contains file paths
func isFilePath(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check if it looks like a file path (has extension or directory structure)
		if strings.Contains(line, "\\") || strings.Contains(line, "/") {
			// Additional check for common file extensions or drive letters
			if strings.Contains(line, ".") || strings.HasPrefix(line, "C:") || strings.HasPrefix(line, "D:") {
				return true
			}
		}
	}
	return false
}
