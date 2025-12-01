package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const testdataDir = "tests/testdata"

func newTestApp() *App {
	return NewApp("host")
}

func mustLoadTestJSON(t *testing.T, filename string, placeholders map[string]string) []byte {
	t.Helper()

	path := filepath.Join(testdataDir, filename)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read testdata %s: %v", filename, err)
	}

	rendered := raw
	for key, value := range placeholders {
		token := []byte("{{" + key + "}}")
		rendered = bytes.ReplaceAll(rendered, token, []byte(value))
	}

	// Remove any remaining tokens so tests fail loudly if we forgot to replace one
	if bytes.Contains(rendered, []byte("{{")) {
		t.Fatalf("testdata %s still contains unreplaced tokens: %s", filename, string(rendered))
	}

	return rendered
}

func decodeJSONBody[T any](t *testing.T, r io.Reader) T {
	t.Helper()
	var out T
	dec := json.NewDecoder(r)
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	return out
}

func decodeResponseBody[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	return decodeJSONBody[T](t, rr.Body)
}
