package main

import (
	"strings"
	"testing"
)

func TestSanitizePlainTextTrimsAndStrips(t *testing.T) {
	input := "  hello\r\nworld\u202e  "
	expected := "hello\nworld"
	if got := sanitizePlainText(input, 0); got != expected {
		t.Fatalf("expected sanitized %q, got %q", expected, got)
	}
}

func TestSanitizePlainTextMaxLen(t *testing.T) {
	input := strings.Repeat("a", maxUserNameLen+10)
	if got := sanitizePlainText(input, maxUserNameLen); len([]rune(got)) != maxUserNameLen {
		t.Fatalf("expected sanitized length %d, got %d", maxUserNameLen, len([]rune(got)))
	}
}

func TestSanitizeClipboardTextUsesPlain(t *testing.T) {
	input := "\tclipboard\rtext"
	expected := "clipboard\ntext"
	if got := sanitizeClipboardText(input); got != expected {
		t.Fatalf("expected sanitized clipboard %q, got %q", expected, got)
	}
}
