package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"GOproject/clip_helper"
)

type mockSSEWriter struct {
	header  http.Header
	buf     bytes.Buffer
	flushed bool
}

func (m *mockSSEWriter) Header() http.Header         { return m.header }
func (m *mockSSEWriter) Write(b []byte) (int, error) { return m.buf.Write(b) }
func (m *mockSSEWriter) WriteHeader(statusCode int)  {}
func (m *mockSSEWriter) Flush()                      { m.flushed = true }

func TestSSEManagerBasics(t *testing.T) {
	sm := NewSSEManager()
	mw := &mockSSEWriter{header: http.Header{}}
	client := sm.AddClient("user1", mw, mw)

	if !sm.IsConnected("user1") {
		t.Fatalf("expected user to be connected")
	}

	if err := sm.SendHeartbeat("user1"); err != nil {
		t.Fatalf("SendHeartbeat error: %v", err)
	}
	if mw.buf.Len() == 0 || !mw.flushed {
		t.Fatalf("expected heartbeat to write and flush")
	}

	sm.RemoveClient(client)
	if sm.IsConnected("user1") {
		t.Fatalf("expected client removed")
	}

	if err := sm.SendToClient("user1", EventChatMessage, "hi"); err == nil {
		t.Fatalf("expected error when sending to disconnected client")
	}
}

func TestCorsMiddleware(t *testing.T) {
	called := false
	h := corsMiddleware(func(w http.ResponseWriter, r *http.Request) { called = true })

	r := httptest.NewRequest("OPTIONS", "/api/test", nil)
	rw := httptest.NewRecorder()
	h(rw, r)
	if rw.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for OPTIONS")
	}
	if called {
		t.Fatalf("handler should not be called for OPTIONS")
	}

	r2 := httptest.NewRequest("GET", "/api/test", nil)
	rw2 := httptest.NewRecorder()
	h(rw2, r2)
	if rw2.Result().StatusCode == 0 {
		t.Fatalf("expected response to be written")
	}
	if !called {
		t.Fatalf("handler should be called for non-OPTIONS")
	}
	if rw2.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("expected CORS headers set")
	}
}

func TestGetEnvDefault(t *testing.T) {
	os.Setenv("TEST_ENV_KEY", "value")
	if v := getEnvDefault("TEST_ENV_KEY", "fallback"); v != "value" {
		t.Fatalf("expected env value, got %s", v)
	}
	os.Unsetenv("TEST_ENV_KEY")
	if v := getEnvDefault("TEST_ENV_KEY", "fallback"); v != "fallback" {
		t.Fatalf("expected fallback when env unset, got %s", v)
	}
}

func TestHumanFileSize(t *testing.T) {
	if v := clip_helper.HumanFileSize(500); v != "500 B" {
		t.Fatalf("unexpected small size: %s", v)
	}
	if v := clip_helper.HumanFileSize(2048); v != "2.0 KB" {
		t.Fatalf("unexpected larger size: %s", v)
	}
}

func TestAppInviteAndJoinRequest(t *testing.T) {
	app := NewApp("host")
	user := app.CreateUser("Bob")
	msg := app.Invite(user.ID)
	if msg == "" {
		t.Fatalf("Invite returned empty message")
	}

	owner := app.CreateUser("Owner")
	room := app.CreateRoom("Room", owner.ID)
	app.users[user.ID] = user
	app.users[owner.ID] = owner

	_, err := app.RequestJoinRoom(user.ID, room.ID)
	if err == nil {
		// Without SSE client, likely returns error; if not, still acceptable
		t.Fatalf("expected error when owner SSE client missing")
	}
}

func TestNetworkSetDisconnected(t *testing.T) {
	nc := NewNetworkClient("http://example.com")
	nc.connected = true
	nc.setDisconnected()
	if nc.IsConnected() {
		t.Fatalf("expected client to be marked disconnected")
	}
}
