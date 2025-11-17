//go:build darwin
// +build darwin

package main

import (
	"context"
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

func main() {
	fmt.Println("Welcome to clipboard testing for MacOS, waiting for response, press \"Cmd+Shift+C\" to copy here.")

	// Watch for clipboard changes
	ch := clipboard.Watch(context.Background(), clipboard.FmtText)
	for data := range ch {
		text := string(data)
		fmt.Printf("You copied: %s\n", text)
	}
}