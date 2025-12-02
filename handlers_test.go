package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	created := decodeResponseBody[*User](t, rr)
	if created.Name != "Test User" {
		t.Fatalf("expected sanitized user name 'Test User', got %s", created.Name)
	}

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

	nameWithNoise := "  Release Room\\r\\n"
	body := bytes.NewReader(mustLoadTestJSON(t, "create_room_template.json", map[string]string{"name": nameWithNoise}))
	req := httptest.NewRequest(http.MethodPost, "/api/rooms", body)
	req.Header.Set("Content-Type", "application/json")
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

	inviterConn := attachClient(app, inviter.ID)
	inviteeConn := attachClient(app, invitee.ID)

	inviteBody := bytes.NewReader(mustLoadTestJSON(t, "invite_request_template.json", map[string]string{
		"userId":    invitee.ID,
		"inviterId": inviter.ID,
		"message":   "Ready to collaborate?",
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/invite", inviteBody)
	req.Header.Set("Content-Type", "application/json")
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
	rr = httptest.NewRecorder()
	app.handleChat(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/chat/{room} expected 200, got %d", rr.Code)
	}

	history := decodeResponseBody[[]*ChatMessage](t, rr)
	if len(history) != 1 || history[0].Message != chatMessage {
		t.Fatalf("expected persisted chat history, got %#v", history)
	}

	ops := app.historyPool.GetOperations(roomID, "")
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
	rr = httptest.NewRecorder()
	app.handleChat(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("second POST /api/chat expected 200, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/operations/"+roomID+"?since="+baseline, nil)
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

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/operations/", nil)
	app.handleOperations(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when room ID missing, got %d", rr.Code)
	}
}
