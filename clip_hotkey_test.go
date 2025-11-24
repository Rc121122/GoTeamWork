//go:build darwin
// +build darwin

package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"golang.design/x/clipboard"
)

// TestStartClipboardHotkey simulates a hotkey press by overriding addEvent.
// It sets the system clipboard, overrides addEvent to return true once,
// and verifies the callback receives the expected ClipboardItem.
func TestStartClipboardHotkey(t *testing.T) {
	// Ensure this test only runs on macOS
	// (build tags at the top already ensure this, but double-check at runtime)

	// Initialize clipboard and set known value
	if err := clipboard.Init(); err != nil {
		t.Fatalf("clipboard init failed: %v", err)
	}
	testText := "hotkey-test-123"
	if err := clipboard.Write(clipboard.FmtText, []byte(testText)); err != nil {
		t.Fatalf("failed to write clipboard: %v", err)
	}

	// Override addEvent to return true exactly once
	origAdd := addEvent
	defer func() { addEvent = origAdd }()
	origMouse := getMousePosition
	defer func() { getMousePosition = origMouse }()

	getMousePosition = func() (int, int) {
		return 125, 260
	}

	var once sync.Once
	addEvent = func(keys ...string) bool {
		called := false
		once.Do(func() { called = true })
		if called {
			return true
		}
		// After first invocation, block briefly until context cancels to simulate single press
		time.Sleep(500 * time.Millisecond)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ch := make(chan struct {
		item *ClipboardItem
		x    int
		y    int
	}, 1)

	if err := StartClipboardHotkey(ctx, func(it *ClipboardItem, x, y int) {
		ch <- struct {
			item *ClipboardItem
			x    int
			y    int
		}{item: it, x: x, y: y}
	}); err != nil {
		t.Fatalf("StartClipboardHotkey returned error: %v", err)
	}

	select {
	case payload := <-ch:
		if payload.item == nil {
			t.Fatalf("received nil clipboard item")
		}
		if payload.item.Type != ClipboardText {
			t.Fatalf("expected clipboard text, got type=%v", payload.item.Type)
		}
		if payload.item.Text != testText {
			t.Fatalf("clipboard text mismatch: got=%q want=%q", payload.item.Text, testText)
		}
		if payload.x != 125 || payload.y != 260 {
			t.Fatalf("unexpected mouse position: got=(%d,%d)", payload.x, payload.y)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for clipboard hotkey callback")
	}
}
