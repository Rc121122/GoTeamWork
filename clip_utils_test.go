package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyClipboardPaths(t *testing.T) {
	tmp := t.TempDir()
	single := filepath.Join(tmp, "file.txt")
	if err := os.WriteFile(single, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write single file: %v", err)
	}

	dir := filepath.Join(tmp, "folder")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir dir: %v", err)
	}

	kind, err := classifyClipboardPaths([]string{single})
	if err != nil || kind != kindSingle {
		t.Fatalf("expected kindSingle, got %v, err %v", kind, err)
	}

	kind, err = classifyClipboardPaths([]string{dir})
	if err != nil || kind != kindDirs {
		t.Fatalf("expected kindDirs, got %v, err %v", kind, err)
	}

	kind, err = classifyClipboardPaths([]string{single, dir})
	if err != nil || kind != kindMixed {
		t.Fatalf("expected kindMixed, got %v, err %v", kind, err)
	}
}

func TestClassifyClipboardPathsError(t *testing.T) {
	if _, err := classifyClipboardPaths([]string{}); err == nil {
		t.Fatalf("expected error for empty paths")
	}
}

func TestTotalClipboardSize(t *testing.T) {
	tmp := t.TempDir()
	file1 := filepath.Join(tmp, "a.bin")
	file2 := filepath.Join(tmp, "b.bin")
	subDir := filepath.Join(tmp, "nested")
	file3 := filepath.Join(subDir, "c.bin")

	if err := os.WriteFile(file1, []byte("abcd"), 0o644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("0123456789"), 0o644); err != nil {
		t.Fatalf("write file2: %v", err)
	}
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if err := os.WriteFile(file3, []byte("xyz"), 0o644); err != nil {
		t.Fatalf("write file3: %v", err)
	}

	total, err := totalClipboardSize([]string{file1, file2, subDir})
	if err != nil {
		t.Fatalf("totalClipboardSize error: %v", err)
	}
	expected := int64(len("abcd") + len("0123456789") + len("xyz"))
	if total != expected {
		t.Fatalf("unexpected total size: %d", total)
	}
}

func TestDetectMimeAndThumb(t *testing.T) {
	if mt := detectMimeType("image.png"); mt != "image/png" {
		t.Fatalf("expected image/png, got %s", mt)
	}
	if mt := detectMimeType("unknownfile"); mt != "application/octet-stream" {
		t.Fatalf("expected fallback mime, got %s", mt)
	}

	if thumb := buildFileThumb("archive.tar.gz"); thumb != "GZ" {
		t.Fatalf("expected thumb GZ, got %s", thumb)
	}
	if thumb := buildFileThumb("verylongname.extended"); len(thumb) > 4 {
		t.Fatalf("thumb should be at most 4 chars, got %s", thumb)
	}
}

func TestSetPendingClipboardFiles(t *testing.T) {
	app := NewApp("host")
	path := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	ok, err := app.SetPendingClipboardFiles([]string{path})
	if err != nil {
		t.Fatalf("SetPendingClipboardFiles error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if app.consumePendingClipboardItem() == nil {
		t.Fatalf("expected cached clipboard item after setting")
	}
}

func TestDecodeBase64(t *testing.T) {
	if _, err := decodeBase64("@@@@"); err == nil {
		t.Fatalf("expected error for invalid base64")
	}
	data, err := decodeBase64("aGVsbG8=")
	if err != nil || string(data) != "hello" {
		t.Fatalf("unexpected decode result: %s, err %v", string(data), err)
	}
}
