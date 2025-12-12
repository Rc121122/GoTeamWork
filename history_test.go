package main

import (
	"fmt"
	"testing"

	"GOproject/clip_helper"
)

func TestHistoryPoolChatAndClipboard(t *testing.T) {
	hp := NewHistoryPool()
	roomID := "room-hist"

	chat := &ChatMessage{ID: "chat_1", RoomID: roomID, UserID: "user_1", UserName: "Alice", Message: "hi", Timestamp: 1}
	hp.AddOperation(roomID, OpAdd, chat.ID, &Item{ID: chat.ID, Type: ItemChat, Data: chat}, "", "")

	clipboard := &clip_helper.ClipboardItem{Type: clip_helper.ClipboardText, Text: "copy"}
	hp.AddOperation(roomID, OpAdd, "clip_1", &Item{ID: "clip_1", Type: ItemClipboard, Data: clipboard}, "", "")

	msgs := hp.GetCurrentChatMessages(roomID)
	if len(msgs) != 1 || msgs[0].ID != chat.ID {
		t.Fatalf("expected chat message to be returned, got %#v", msgs)
	}

	items := hp.GetCurrentClipboardItems(roomID)
	if len(items) != 1 || items[0].Text != clipboard.Text {
		t.Fatalf("expected clipboard item to be returned, got %#v", items)
	}

	hp.AddOperation(roomID, OpRemove, chat.ID, &Item{ID: chat.ID, Type: ItemChat}, "", "")
	msgs = hp.GetCurrentChatMessages(roomID)
	if len(msgs) != 0 {
		t.Fatalf("expected chat removal to be reflected, got %#v", msgs)
	}
}

func TestHistoryPoolEnforceLimits(t *testing.T) {
	hp := NewHistoryPool()
	roomID := "room-large"

	for i := 0; i < maxOperationsPerRoom+10; i++ {
		id := fmt.Sprintf("msg_%d", i)
		msg := &ChatMessage{ID: id, RoomID: roomID, UserID: "user", UserName: "User", Message: id, Timestamp: int64(i)}
		hp.AddOperation(roomID, OpAdd, id, &Item{ID: id, Type: ItemChat, Data: msg}, "", "")
	}

	ops := hp.GetOperations(roomID, "", "")
	if len(ops) != maxOperationsPerRoom {
		t.Fatalf("expected history trimmed to %d ops, got %d", maxOperationsPerRoom, len(ops))
	}

	if ops[0].ID == "msg_0" {
		t.Fatalf("oldest operations should have been trimmed")
	}
}
