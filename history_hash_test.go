package main

import (
	"testing"
)

// Verifies operation hashes form a linear chain and parent hashes link correctly.
func TestHistoryHashChain(t *testing.T) {
	hp := NewHistoryPool()
	room := "room-hash"

	op1 := hp.AddOperation(room, OpAdd, "item1", &Item{ID: "item1", Type: ItemChat, Data: &ChatMessage{ID: "item1", Message: "hi", Timestamp: 1}}, "u1", "Alice")
	op2 := hp.AddOperation(room, OpAdd, "item2", &Item{ID: "item2", Type: ItemChat, Data: &ChatMessage{ID: "item2", Message: "yo", Timestamp: 2}}, "u2", "Bob")

	if op1.Hash == "" || op2.Hash == "" {
		t.Fatalf("expected hashes to be populated")
	}
	if op2.ParentHash != op1.Hash {
		t.Fatalf("expected op2 parent hash %s, got %s", op1.Hash, op2.ParentHash)
	}
}

// Ensures incremental fetch by hash returns only newer operations and falls back when hash is unknown.
func TestGetOperationsSinceHash(t *testing.T) {
	hp := NewHistoryPool()
	room := "room-hash-since"

	op1 := hp.AddOperation(room, OpAdd, "item1", &Item{ID: "item1", Type: ItemChat, Data: &ChatMessage{ID: "item1", Message: "a", Timestamp: 1}}, "u1", "Alice")
	op2 := hp.AddOperation(room, OpAdd, "item2", &Item{ID: "item2", Type: ItemChat, Data: &ChatMessage{ID: "item2", Message: "b", Timestamp: 2}}, "u1", "Alice")

	ops := hp.GetOperations(room, "", op1.Hash)
	if len(ops) != 1 || ops[0].ID != op2.ID {
		t.Fatalf("expected only op2 after sinceHash, got %+v", ops)
	}

	// Unknown hash should return full history for resync
	full := hp.GetOperations(room, "", "unknown")
	if len(full) != 2 {
		t.Fatalf("expected full history on unknown hash, got %d", len(full))
	}

	// SinceID path still works
	byID := hp.GetOperations(room, op1.ID, "")
	if len(byID) != 1 || byID[0].ID != op2.ID {
		t.Fatalf("expected op2 after sinceID, got %+v", byID)
	}
}
