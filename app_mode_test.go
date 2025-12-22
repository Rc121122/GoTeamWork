package main

import "testing"

func TestSetModeClientInitialization(t *testing.T) {
	app := NewApp("pending")

	if app.Mode != "pending" {
		t.Fatalf("expected initial mode pending, got %s", app.Mode)
	}

	mode, err := app.SetMode("client")
	if err != nil {
		t.Fatalf("SetMode client returned error: %v", err)
	}
	if mode != "client" || app.Mode != "client" {
		t.Fatalf("expected mode client, got %s", mode)
	}
	if !app.modeInitialized {
		t.Fatalf("mode should be initialized after SetMode")
	}
	if app.networkClient == nil {
		t.Fatalf("networkClient should be initialized for client mode")
	}
}

func TestSetModeInvalid(t *testing.T) {
	app := NewApp("pending")
	if _, err := app.SetMode("invalid"); err == nil {
		t.Fatalf("expected error for invalid mode")
	}
}

func TestSetModeAlreadyInitialized(t *testing.T) {
	app := NewApp("pending")
	if _, err := app.SetMode("client"); err != nil {
		t.Fatalf("unexpected error on first set: %v", err)
	}
	if _, err := app.SetMode("host"); err == nil {
		t.Fatalf("expected error when switching mode after initialization")
	}
}

func TestSetServerURLAndConnectionStatus(t *testing.T) {
	app := NewApp("client")
	if _, err := app.SetMode("client"); err != nil {
		t.Fatalf("unexpected error on SetMode: %v", err)
	}

	app.SetServerURL("http://localhost:8080")
	if app.networkClient == nil || app.networkClient.serverURL != "http://localhost:8080" {
		t.Fatalf("server URL not set correctly")
	}

	if app.GetConnectionStatus() {
		t.Fatalf("expected disconnected status initially")
	}
	app.networkClient.connected = true
	if !app.GetConnectionStatus() {
		t.Fatalf("expected connected status after flag set")
	}
}

func TestGreet(t *testing.T) {
	app := NewApp("pending")
	got := app.Greet("Tester")
	if got == "" {
		t.Fatalf("greet should return non-empty string")
	}
}
