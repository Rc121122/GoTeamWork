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
	maxInviteMessageLen = 280
)

// sanitizePlainText trims whitespace, normalizes CRLF -> LF, strips control/format/private/non-character runes,
// and enforces a maximum rune length so hostile payloads cannot overflow downstream buffers.
func sanitizePlainText(input string, maxLen int) string {
	// Normalize CRLF and stray CR to LF
	s := strings.ReplaceAll(input, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Trim surrounding whitespace
	s = strings.TrimSpace(s)

	cleaned := strings.Map(func(r rune) rune {
		// Remove Cc (control) except LF and TAB
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		// Remove format characters (zero-width, bidi overrides, etc.)
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		// Remove surrogate, private-use
		if unicode.Is(unicode.Cs, r) || unicode.Is(unicode.Co, r) {
			return -1
		}
		// Remove Unicode non-characters U+FDD0..U+FDEF and code points ending with FFFE/FFFF
		if r >= 0xFDD0 && r <= 0xFDEF {
			return -1
		}
		if r&0xFFFE == 0xFFFE {
			return -1
		}
		return r
	}, s)

	// Enforce maxLen on rune count (consider grapheme-cluster-based truncation if needed)
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

func sanitizeInviteMessage(msg string) string {
	return sanitizePlainText(msg, maxInviteMessageLen)
}

func sanitizeClipboardText(text string) string {
	return sanitizePlainText(text, maxClipboardTextLen)
}
