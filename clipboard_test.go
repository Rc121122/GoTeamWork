package main

import "testing"

func TestHandleClipboardCopySanitizesAndBroadcasts(t *testing.T) {
	app := newTestApp()
	listener := attachClient(app, "listener")

	raw := &ClipboardItem{Type: ClipboardText, Text: "  \r\n\u202eShared "}
	app.handleClipboardCopy(raw)

	events := listener.Events()
	evt, ok := findEvent(events, EventClipboardCopied)
	if !ok {
		t.Fatalf("expected clipboard_copied SSE, got %#v", events)
	}

	payload := decodeEventPayload[ClipboardItem](t, evt)
	if payload.Text != "Shared" {
		t.Fatalf("expected sanitized clipboard text 'Shared', got %q", payload.Text)
	}

	ops := app.historyPool.GetOperations("global", "", "")
	if len(ops) != 1 {
		t.Fatalf("expected clipboard history stored in global room, got %d operations", len(ops))
	}
}

func TestHandleClipboardCopyNoItem(t *testing.T) {
	app := newTestApp()
	listener := attachClient(app, "listener")

	app.handleClipboardCopy(nil)

	if events := listener.Events(); len(events) != 0 {
		t.Fatalf("expected no SSE events when clipboard item is nil, got %#v", events)
	}
}
