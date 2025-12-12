package main

import (
	"strings"
	"testing"
	"time"
)

func TestSSERouteNewUserBroadcast(t *testing.T) {
	app := NewApp("host")
	observer := attachClient(app, "observer")

	user := app.CreateUser("Delta")
	if user == nil {
		t.Fatalf("expected user creation to succeed")
	}

	events := observer.Events()
	evt, found := findEvent(events, EventUserCreated)
	if !found {
		t.Fatalf("expected user_created event, got %#v", events)
	}

	payload := decodeEventPayload[User](t, evt)
	if payload.Name != user.Name {
		t.Fatalf("expected payload name %s, got %s", user.Name, payload.Name)
	}
	if payload.ID != user.ID {
		t.Fatalf("expected payload id %s, got %s", user.ID, payload.ID)
	}
}

func TestSSERouteInviteMessageLeave(t *testing.T) {
	app := NewApp("host")
	inviter := app.CreateUser("Alice")
	invitee := app.CreateUser("Bob")

	inviterConn := attachClient(app, inviter.ID)
	inviteeConn := attachClient(app, invitee.ID)

	inviteID, inviteMessage, expiresAt := app.InviteWithRoom(invitee.ID, inviter.ID, "Let's pair up")
	if inviteID == "" || strings.HasPrefix(inviteMessage, "Error") {
		t.Fatalf("expected invite to succeed, got invite=%s message=%s", inviteID, inviteMessage)
	}
	if expiresAt == 0 {
		t.Fatalf("expected expiry timestamp from InviteWithRoom")
	}

	inviteEvents := inviteeConn.Events()
	evt, found := findEvent(inviteEvents, EventUserInvited)
	if !found {
		t.Fatalf("expected user_invited event for invitee, got %#v", inviteEvents)
	}

	type invitePayload struct {
		InviteID string `json:"inviteId"`
		Inviter  string `json:"inviter"`
		Message  string `json:"message"`
	}
	invite := decodeEventPayload[invitePayload](t, evt)
	if invite.InviteID != inviteID {
		t.Fatalf("expected invite payload id %s, got %s", inviteID, invite.InviteID)
	}

	inviterConn.Reset()
	inviteeConn.Reset()

	roomID, acceptMessage := app.AcceptInvite(inviteID, invitee.ID)
	if strings.HasPrefix(acceptMessage, "Error") {
		t.Fatalf("expected accept to succeed, got %s", acceptMessage)
	}
	if roomID == "" {
		t.Fatalf("expected room ID from accept flow")
	}

	inviterJoinedEvents := inviterConn.Events()

	type joinedPayload struct {
		RoomID   string `json:"roomId"`
		RoomName string `json:"roomName"`
		UserID   string `json:"userId"`
		UserName string `json:"userName"`
	}

	foundInvitee := false
	for _, evt := range inviterJoinedEvents {
		if evt.Name != string(EventUserJoined) {
			continue
		}
		payload := decodeEventPayload[joinedPayload](t, evt)
		if payload.UserID == invitee.ID {
			foundInvitee = true
			break
		}
	}

	if !foundInvitee {
		t.Fatalf("expected inviter to receive invitee join event, got %#v", inviterJoinedEvents)
	}

	inviteeJoinedEvents := inviteeConn.Events()
	if _, ok := findEvent(inviteeJoinedEvents, EventUserJoined); !ok {
		t.Fatalf("expected user_joined notification for joiner, got %#v", inviteeJoinedEvents)
	}

	inviterConn.Reset()
	inviteeConn.Reset()

	chatMessage := app.SendChatMessage(roomID, inviter.ID, "Hello team")
	if strings.HasPrefix(chatMessage, "Error") {
		t.Fatalf("expected chat to succeed, got %s", chatMessage)
	}

	chatEvents := inviteeConn.Events()
	chatEvt, found := findEvent(chatEvents, EventChatMessage)
	if !found {
		t.Fatalf("expected chat_message for invitee, got %#v", chatEvents)
	}

	msg := decodeEventPayload[ChatMessage](t, chatEvt)
	if msg.Message != "Hello team" {
		t.Fatalf("expected chat payload 'Hello team', got %s", msg.Message)
	}

	if events := inviterConn.Events(); len(events) != 0 {
		t.Fatalf("sender should not receive chat event, got %#v", events)
	}

	inviterConn.Reset()
	inviteeConn.Reset()

	leaveMessage := app.LeaveRoom(invitee.ID)
	if strings.HasPrefix(leaveMessage, "Error") {
		t.Fatalf("expected leave to succeed, got %s", leaveMessage)
	}

	inviterEvents := inviterConn.Events()
	leaveEvt, found := findEvent(inviterEvents, EventUserLeft)
	if !found {
		t.Fatalf("expected user_left event for remaining member, got %#v", inviterEvents)
	}

	left := decodeEventPayload[joinedPayload](t, leaveEvt)
	if left.UserID != invitee.ID {
		t.Fatalf("expected leave payload user %s, got %s", invitee.ID, left.UserID)
	}

	deleteEvt, found := findEvent(inviterEvents, EventRoomDeleted)
	if !found {
		t.Fatalf("expected room_deleted event for remaining member, got %#v", inviterEvents)
	}

	type roomPayload struct {
		RoomID   string `json:"roomId"`
		RoomName string `json:"roomName"`
	}
	deleted := decodeEventPayload[roomPayload](t, deleteEvt)
	if deleted.RoomID != roomID {
		t.Fatalf("expected deleted room %s, got %s", roomID, deleted.RoomID)
	}

	if _, exists := app.rooms[roomID]; exists {
		t.Fatalf("room %s should have been cleaned up", roomID)
	}
}

func TestSSERouteReconnectHistory(t *testing.T) {
	history := NewHistoryPool()
	roomID := "room_reconnect"

	itemOne := &Item{
		ID:   "msg_1",
		Type: ItemChat,
		Data: &ChatMessage{
			ID:        "msg_1",
			RoomID:    roomID,
			UserID:    "user_1",
			UserName:  "Alice",
			Message:   "First",
			Timestamp: time.Now().Unix(),
		},
	}
	op1 := history.AddOperation(roomID, OpAdd, itemOne.ID, itemOne, "", "")

	itemTwo := &Item{
		ID:   "msg_2",
		Type: ItemChat,
		Data: &ChatMessage{
			ID:        "msg_2",
			RoomID:    roomID,
			UserID:    "user_2",
			UserName:  "Bob",
			Message:   "Second",
			Timestamp: time.Now().Unix(),
		},
	}
	op2 := history.AddOperation(roomID, OpAdd, itemTwo.ID, itemTwo, "", "")

	if ops := history.GetOperations(roomID, "", ""); len(ops) != 2 {
		t.Fatalf("expected 2 operations for full sync, got %d", len(ops))
	}

	opsSince := history.GetOperations(roomID, op1.ID, "")
	if len(opsSince) != 1 {
		t.Fatalf("expected 1 operation since op1, got %d", len(opsSince))
	}
	if opsSince[0].ID != op2.ID {
		t.Fatalf("expected next op to be %s, got %s", op2.ID, opsSince[0].ID)
	}

	if ops := history.GetOperations(roomID, "missing", ""); len(ops) != 0 {
		t.Fatalf("expected empty result for unknown hash, got %d", len(ops))
	}
}
