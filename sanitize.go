package main

import (
	"strings"
	"unicode"
)

const (
	maxUserNameLen      = 32
	maxRoomNameLen      = 64
	maxChatMessageLen   = 2000
	maxClipboardTextLen = 4000
)

// sanitizePlainText trims whitespace, strips control characters (except standard newlines/tabs),
// and enforces a maximum rune length so hostile payloads cannot overflow downstream buffers.
func sanitizePlainText(input string, maxLen int) string {
	trimmed := strings.TrimSpace(input)

	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, trimmed)

	if maxLen > 0 {
		runes := []rune(cleaned)
		if len(runes) > maxLen {
			cleaned = string(runes[:maxLen])
		}
	}

	return cleaned
}

func sanitizeUserName(name string) string {
	return sanitizePlainText(name, maxUserNameLen)
}

func sanitizeRoomName(name string) string {
	return sanitizePlainText(name, maxRoomNameLen)
}

func sanitizeChatMessage(msg string) string {
	return sanitizePlainText(msg, maxChatMessageLen)
}

func sanitizeClipboardText(text string) string {
	return sanitizePlainText(text, maxClipboardTextLen)
}
