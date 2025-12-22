package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestSetUserAndGetMode(t *testing.T) {
	app := NewApp("host")
	user := app.SetUser("u1", "Alice")
	if app.currentUser == nil || app.currentUser.ID != "u1" || app.users[user.ID] == nil {
		t.Fatalf("SetUser did not update current user and map")
	}
	if app.GetMode() != "host" {
		t.Fatalf("expected mode host, got %s", app.GetMode())
	}
}

func TestGetCurrentRoom(t *testing.T) {
	app := NewApp("host")
	room := app.CreateRoom("My Room", "host")
	app.currentRoom = room
	if got := app.GetCurrentRoom(); got == nil || got.ID != room.ID {
		t.Fatalf("GetCurrentRoom returned wrong room")
	}
}

func TestSaveDroppedFiles(t *testing.T) {
	app := NewApp("host")
	content := base64.StdEncoding.EncodeToString([]byte("hello"))
	files, err := app.SaveDroppedFiles([]DroppedFilePayload{{Name: "test.txt", Data: content}})
	if err != nil {
		t.Fatalf("SaveDroppedFiles error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 top path, got %d", len(files))
	}
	if _, err := os.Stat(files[0]); err != nil {
		t.Fatalf("written file missing: %v", err)
	}

	// ensure relative path handling
	content2 := base64.StdEncoding.EncodeToString([]byte("world"))
	files2, err := app.SaveDroppedFiles([]DroppedFilePayload{{Name: "nested.txt", Rel: "folder/sub", Data: content2}})
	if err != nil {
		t.Fatalf("SaveDroppedFiles nested error: %v", err)
	}
	if len(files2) != 1 {
		t.Fatalf("expected 1 root for nested path, got %d", len(files2))
	}
	if _, err := os.Stat(filepath.Join(files2[0], "sub", "nested.txt")); err != nil {
		t.Fatalf("nested file missing: %v", err)
	}
}
