package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type mockSSEConn struct {
	header  http.Header
	buf     bytes.Buffer
	status  int
	flushes int
}

func newMockSSEConn() *mockSSEConn {
	return &mockSSEConn{header: make(http.Header)}
}

func (m *mockSSEConn) Header() http.Header {
	return m.header
}

func (m *mockSSEConn) Write(b []byte) (int, error) {
	return m.buf.Write(b)
}

func (m *mockSSEConn) WriteHeader(statusCode int) {
	m.status = statusCode
}

func (m *mockSSEConn) Flush() {
	m.flushes++
}

func (m *mockSSEConn) Reset() {
	m.buf.Reset()
	m.flushes = 0
}

type recordedEvent struct {
	Name string
	Data string
}

func (m *mockSSEConn) Events() []recordedEvent {
	raw := strings.TrimSpace(m.buf.String())
	if raw == "" {
		return nil
	}

	blocks := strings.Split(raw, "\n\n")
	events := make([]recordedEvent, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		var evt recordedEvent
		for _, line := range lines {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "event:"):
				evt.Name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				evt.Data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
		}

		if evt.Name != "" {
			events = append(events, evt)
		}
	}

	return events
}

func attachClient(app *App, userID string) *mockSSEConn {
	conn := newMockSSEConn()
	app.sseManager.AddClient(userID, conn, conn)
	return conn
}

func findEvent(events []recordedEvent, eventType SSEEventType) (recordedEvent, bool) {
	for _, evt := range events {
		if evt.Name == string(eventType) {
			return evt, true
		}
	}
	return recordedEvent{}, false
}

type sseEnvelope struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
}

func decodeEventPayload[T any](t *testing.T, evt recordedEvent) T {
	t.Helper()

	var envelope sseEnvelope
	if err := json.Unmarshal([]byte(evt.Data), &envelope); err != nil {
		t.Fatalf("failed to decode SSE envelope %s: %v", evt.Name, err)
	}
	if envelope.Type != evt.Name {
		t.Fatalf("envelope type %s mismatch header %s", envelope.Type, evt.Name)
	}

	var payload T
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		t.Fatalf("failed to decode payload for %s: %v", evt.Name, err)
	}
	return payload
}
