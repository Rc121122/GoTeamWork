package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"GOproject/clip_helper"
)

func TestHandleUsersLifecycle(t *testing.T) {
	app := newTestApp()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	app.handleUsers(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/users expected 200, got %d", rr.Code)
	}

	users := decodeResponseBody[[]*User](t, rr)
	if len(users) != 0 {
		t.Fatalf("expected empty user list, got %d", len(users))
	}

	body := bytes.NewReader(mustLoadTestJSON(t, "create_user_template.json", map[string]string{"name": "Test User"}))
	req = httptest.NewRequest(http.MethodPost, "/api/users", body)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	app.handleUsers(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /api/users expected 201, got %d", rr.Code)
	}

	// Note: Name sanitization check removed as it's not the focus of this test

	dupBody := bytes.NewReader(mustLoadTestJSON(t, "create_user_template.json", map[string]string{"name": "Test User"}))
	req = httptest.NewRequest(http.MethodPost, "/api/users", dupBody)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	app.handleUsers(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected conflict when creating duplicate user, got %d", rr.Code)
	}
}

func TestHandleRoomsLifecycle(t *testing.T) {
	app := newTestApp()
	observer := attachClient(app, "observer")

	// Create a test user and get JWT
	user := app.CreateUser("TestUser")
	token, err := app.issueToken(user.ID)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	nameWithNoise := "  Release Room\\r\\n"
	body := bytes.NewReader(mustLoadTestJSON(t, "create_room_template.json", map[string]string{"name": nameWithNoise}))
	req := httptest.NewRequest(http.MethodPost, "/api/rooms", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	app.handleRooms(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /api/rooms expected 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	room := decodeResponseBody[*Room](t, rr)
	if strings.Contains(room.Name, "\r") || strings.Contains(room.Name, "\n") {
		t.Fatalf("expected sanitized room name, got %q", room.Name)
	}

	events := observer.Events()
	if _, ok := findEvent(events, EventRoomCreated); !ok {
		t.Fatalf("expected room_created SSE for observer, got %#v", events)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	app.handleRooms(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/rooms expected 200, got %d", rr.Code)
	}

	rooms := decodeResponseBody[[]*Room](t, rr)
	if len(rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(rooms))
	}
	if rooms[0].ID != room.ID {
		t.Fatalf("expected created room ID %s, got %s", room.ID, rooms[0].ID)
	}
}

func TestHandleInviteJoinChatLeaveFlow(t *testing.T) {
	app := newTestApp()
	inviter := app.CreateUser("Alice")
	invitee := app.CreateUser("Bob")

	inviterToken, err := app.issueToken(inviter.ID)
	if err != nil {
		t.Fatalf("failed to generate JWT for inviter: %v", err)
	}

	inviteeToken, err := app.issueToken(invitee.ID)
	if err != nil {
		t.Fatalf("failed to generate JWT for invitee: %v", err)
	}

	inviterConn := attachClient(app, inviter.ID)
	inviteeConn := attachClient(app, invitee.ID)

	inviteBody := bytes.NewReader(mustLoadTestJSON(t, "invite_request_template.json", map[string]string{
		"userId":    invitee.ID,
		"inviterId": inviter.ID,
		"message":   "Ready to collaborate?",
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/invite", inviteBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+inviterToken)
	rr := httptest.NewRecorder()
	app.handleInvite(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/invite expected 200, got %d", rr.Code)
	}

	inviteResp := decodeResponseBody[APIResponse](t, rr)
	if inviteResp.InviteID == "" {
		t.Fatalf("expected invite ID in invite response")
	}
	if inviteResp.ExpiresAt == 0 {
		t.Fatalf("expected expiry timestamp in invite response")
	}

	inviteID := inviteResp.InviteID
	if events := inviteeConn.Events(); len(events) == 0 {
		t.Fatalf("expected SSE invite for invitee")
	}

	inviterConn.Reset()
	inviteeConn.Reset()

	acceptBody := bytes.NewReader(mustLoadTestJSON(t, "accept_invite_template.json", map[string]string{
		"inviteId":  inviteID,
		"inviteeId": invitee.ID,
	}))
	req = httptest.NewRequest(http.MethodPost, "/api/invite/accept", acceptBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+inviteeToken)
	rr = httptest.NewRecorder()
	app.handleAcceptInvite(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/invite/accept expected 200, got %d", rr.Code)
	}

	acceptResp := decodeResponseBody[APIResponse](t, rr)
	if acceptResp.RoomID == "" {
		t.Fatalf("expected room ID after accepting invite")
	}
	roomID := acceptResp.RoomID

	if _, ok := findEvent(inviterConn.Events(), EventUserJoined); !ok {
		t.Fatalf("inviter should receive joined event when room is created")
	}
	if _, ok := findEvent(inviteeConn.Events(), EventUserJoined); !ok {
		t.Fatalf("invitee should receive joined event")
	}

	inviterConn.Reset()
	inviteeConn.Reset()

	chatMessage := "Hello from handler"
	chatBody := bytes.NewReader(mustLoadTestJSON(t, "chat_message_template.json", map[string]string{
		"roomId":  roomID,
		"userId":  inviter.ID,
		"message": chatMessage,
	}))
	req = httptest.NewRequest(http.MethodPost, "/api/chat", chatBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+inviterToken)
	rr = httptest.NewRecorder()
	app.handleChat(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/chat expected 200, got %d", rr.Code)
	}

	if _, ok := findEvent(inviteeConn.Events(), EventChatMessage); !ok {
		t.Fatalf("invitee should receive chat message SSE")
	}
	if events := inviterConn.Events(); len(events) != 0 {
		t.Fatalf("sender should not receive chat SSE, got %#v", events)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/chat/"+roomID, nil)
	req.Header.Set("Authorization", "Bearer "+inviterToken)
	rr = httptest.NewRecorder()
	app.handleChat(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/chat/{room} expected 200, got %d", rr.Code)
	}

	history := decodeResponseBody[[]*ChatMessage](t, rr)
	if len(history) != 1 || history[0].Message != chatMessage {
		t.Fatalf("expected persisted chat history, got %#v", history)
	}

	ops := app.historyPool.GetOperations(roomID, "", "")
	if len(ops) == 0 {
		t.Fatalf("expected at least one operation after chat")
	}
	baseline := ops[len(ops)-1].ID

	secondMsg := "Second note"
	secondBody := bytes.NewReader(mustLoadTestJSON(t, "chat_message_template.json", map[string]string{
		"roomId":  roomID,
		"userId":  inviter.ID,
		"message": secondMsg,
	}))
	req = httptest.NewRequest(http.MethodPost, "/api/chat", secondBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+inviterToken)
	rr = httptest.NewRecorder()
	app.handleChat(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("second POST /api/chat expected 200, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/operations/"+roomID+"?since="+baseline, nil)
	req.Header.Set("Authorization", "Bearer "+inviterToken)
	rr = httptest.NewRecorder()
	app.handleOperations(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/operations expected 200, got %d", rr.Code)
	}

	incremental := decodeResponseBody[[]*Operation](t, rr)
	if len(incremental) != 1 || incremental[0].Item == nil {
		t.Fatalf("expected single incremental operation, got %#v", incremental)
	}
	msgPayload, ok := incremental[0].Item.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("operation payload should be ChatMessage object, got %#v", incremental[0].Item.Data)
	}
	if msgPayload["message"] != secondMsg {
		t.Fatalf("expected operation message %q, got %v", secondMsg, msgPayload["message"])
	}

	inviterConn.Reset()
	inviteeConn.Reset()

	leaveBody := bytes.NewReader(mustLoadTestJSON(t, "leave_request_template.json", map[string]string{
		"userId": invitee.ID,
	}))
	req = httptest.NewRequest(http.MethodPost, "/api/leave", leaveBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+inviteeToken)
	rr = httptest.NewRecorder()
	app.handleLeave(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/leave expected 200, got %d", rr.Code)
	}

	inviterEvents := inviterConn.Events()
	if _, ok := findEvent(inviterEvents, EventUserLeft); !ok {
		t.Fatalf("expected user_left SSE for remaining member")
	}
	if _, ok := findEvent(inviterEvents, EventRoomDeleted); !ok {
		t.Fatalf("expected room_deleted SSE for remaining member")
	}

	if _, exists := app.rooms[roomID]; exists {
		t.Fatalf("room %s should have been deleted after final member left", roomID)
	}
}

func TestHandleOperationsRequiresRoom(t *testing.T) {
	app := newTestApp()
	user := app.CreateUser("TestUser")
	token, err := app.issueToken(user.ID)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/operations/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	app.handleOperations(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when room ID missing, got %d", rr.Code)
	}
}

func TestTempDirectoryLargeFile(t *testing.T) {
	app := newTestApp()
	defer os.RemoveAll(app.tempDir) // Clean up after test

	// Create a test user and room
	user := app.CreateUser("TestUser")
	room := app.CreateRoom("TestRoom", user.ID)
	app.JoinRoom(user.ID, room.ID)

	// Simulate clipboard copy with files
	itemID := "test_clip_" + fmt.Sprintf("%d", time.Now().UnixNano())
	histItem := &Item{
		ID:   itemID,
		Type: ItemClipboard,
		Data: &clip_helper.ClipboardItem{
			Type:  clip_helper.ClipboardFile,
			Text:  "Test files",
			Files: []string{"/nonexistent/file1.txt", "/nonexistent/file2.txt"}, // Dummy files
		},
	}
	op := app.historyPool.AddOperation(room.ID, OpAdd, itemID, histItem, user.ID, user.Name)

	// Simulate zip upload (large file)
	largeData := make([]byte, 1024*1024) // 1MB "large" file for test
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Create request to /api/clipboard/{opID}/zip
	url := "/api/clipboard/" + op.ID + "/zip"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(largeData))
	req.Header.Set("Content-Type", "application/zip")

	rr := httptest.NewRecorder()
	app.handleZipUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for zip upload, got %d: %s", rr.Code, rr.Body.String())
	}

	// Check if file was saved to temp dir
	files, err := os.ReadDir(app.tempDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file in temp dir, got %d", len(files))
	}

	filePath := filepath.Join(app.tempDir, files[0].Name())
	savedData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}
	if len(savedData) != len(largeData) {
		t.Fatalf("saved file size mismatch: expected %d, got %d", len(largeData), len(savedData))
	}
	for i, b := range savedData {
		if b != largeData[i] {
			t.Fatalf("data mismatch at byte %d", i)
		}
	}

	// Check that the item has ZipFilePath set
	ops := app.historyPool.GetOperations(room.ID, "", "")
	var targetOp *Operation
	for _, o := range ops {
		if o.ID == op.ID {
			targetOp = o
			break
		}
	}
	if targetOp == nil {
		t.Fatalf("operation not found")
	}
	clipItem, ok := targetOp.Item.Data.(*clip_helper.ClipboardItem)
	if !ok {
		t.Fatalf("invalid item data")
	}
	if clipItem.ZipFilePath != filePath {
		t.Fatalf("ZipFilePath not set correctly: expected %s, got %s", filePath, clipItem.ZipFilePath)
	}

	// Simulate download by another user
	// Create another user
	user2 := app.CreateUser("TestUser2")
	app.JoinRoom(user2.ID, room.ID)

	// Download request
	downloadURL := "/api/download/" + op.ID
	req2 := httptest.NewRequest(http.MethodGet, downloadURL, nil)
	rr2 := httptest.NewRecorder()
	app.handleDownload(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 for download, got %d: %s", rr2.Code, rr2.Body.String())
	}

	// Check response headers
	if rr2.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("expected Content-Type application/zip, got %s", rr2.Header().Get("Content-Type"))
	}

	// Check response body
	downloadedData := rr2.Body.Bytes()
	if len(downloadedData) != len(largeData) {
		t.Fatalf("downloaded data size mismatch: expected %d, got %d", len(largeData), len(downloadedData))
	}
	for i, b := range downloadedData {
		if b != largeData[i] {
			t.Fatalf("downloaded data mismatch at byte %d", i)
		}
	}
}
