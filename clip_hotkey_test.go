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
    orig := addEvent
    defer func() { addEvent = orig }()

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

    ch := make(chan *ClipboardItem, 1)

    if err := StartClipboardHotkey(ctx, func(it *ClipboardItem) { ch <- it }); err != nil {
        t.Fatalf("StartClipboardHotkey returned error: %v", err)
    }

    select {
    case it := <-ch:
        if it == nil {
            t.Fatalf("received nil clipboard item")
        }
        if it.Type != ClipboardText {
            t.Fatalf("expected clipboard text, got type=%v", it.Type)
        }
        if it.Text != testText {
            t.Fatalf("clipboard text mismatch: got=%q want=%q", it.Text, testText)
        }
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout waiting for clipboard hotkey callback")
    }
}
