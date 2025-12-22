package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNetworkClientConnectAndCreateUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users":
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				return
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				resp := User{ID: "user_test", Name: "Tester"}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
		case "/api/invite":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	nc := NewNetworkClient(ts.URL)
	if err := nc.ConnectToServer(); err != nil {
		t.Fatalf("ConnectToServer failed: %v", err)
	}
	if !nc.IsConnected() {
		t.Fatalf("expected connected after ConnectToServer")
	}

	user, err := nc.CreateUser("Tester")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.ID != "user_test" || user.Name != "Tester" {
		t.Fatalf("unexpected user response: %+v", user)
	}

	msg, err := nc.SendInvite("invitee", "inviter", "hi")
	if err != nil || msg != "ok" {
		t.Fatalf("SendInvite unexpected result: %s, err %v", msg, err)
	}
}

func TestNetworkClientConnectFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	nc := NewNetworkClient(ts.URL)
	if err := nc.ConnectToServer(); err == nil {
		t.Fatalf("expected failure when server returns 503")
	}
	if nc.IsConnected() {
		t.Fatalf("expected not connected after failure")
	}
}

func TestNetworkClientUploadsAndPing(t *testing.T) {
	zipHit := 0
	singleHit := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/users":
			w.WriteHeader(http.StatusOK)
			return
		case strings.HasPrefix(r.URL.Path, "/api/clipboard/op123/zip"):
			if r.URL.Query().Get("single") == "1" {
				singleHit++
			} else {
				zipHit++
			}
			w.WriteHeader(http.StatusOK)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	nc := NewNetworkClient(ts.URL)

	if err := nc.UploadZipData("op123", []byte("zipdata")); err != nil {
		t.Fatalf("UploadZipData error: %v", err)
	}

	tmpDir := t.TempDir()
	zipFile := filepath.Join(tmpDir, "archive.tar")
	if err := os.WriteFile(zipFile, []byte("tarcontent"), 0o644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}
	if err := nc.UploadZipFile("op123", zipFile); err != nil {
		t.Fatalf("UploadZipFile error: %v", err)
	}

	if err := nc.UploadSingleFileData("op123", []byte("hello"), "file.txt", "text/plain", 5, "TXT"); err != nil {
		t.Fatalf("UploadSingleFileData error: %v", err)
	}

	singleFile := filepath.Join(tmpDir, "single.bin")
	if err := os.WriteFile(singleFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("write single file: %v", err)
	}
	if err := nc.UploadSingleFile("op123", singleFile, "single.bin", "application/octet-stream", 4, "BIN"); err != nil {
		t.Fatalf("UploadSingleFile error: %v", err)
	}

	if err := nc.Ping(); err != nil {
		t.Fatalf("Ping error: %v", err)
	}

	if zipHit == 0 || singleHit == 0 {
		t.Fatalf("expected both zip and single upload endpoints to be hit; zip %d single %d", zipHit, singleHit)
	}
}
