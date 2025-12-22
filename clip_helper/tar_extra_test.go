package clip_helper

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTarPathsWithProgress(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "sample.txt")
	if err := os.WriteFile(filePath, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var progressCalled bool
	var buf bytes.Buffer
	if err := TarPathsWithProgress([]string{filePath}, &buf, func(_, _ int64, _ string) { progressCalled = true }); err != nil {
		t.Fatalf("TarPathsWithProgress error: %v", err)
	}

	tr := tar.NewReader(bytes.NewReader(buf.Bytes()))
	hdr, err := tr.Next()
	if err != nil {
		t.Fatalf("tar read error: %v", err)
	}
	if hdr.Name == "" {
		t.Fatalf("expected header name")
	}
	data, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("read tar payload: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("unexpected tar content: %s", string(data))
	}
	if !progressCalled {
		t.Fatalf("expected progress callback to be invoked")
	}
}

func TestTarPathsFast(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "fast.txt")
	if err := os.WriteFile(filePath, []byte("fast"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var buf bytes.Buffer
	if err := TarPathsFast([]string{filePath}, &buf); err == nil {
		if buf.Len() == 0 {
			t.Fatalf("expected tar data from TarPathsFast")
		}
	} else if err != nil && !strings.Contains(err.Error(), "file already exists") {
		t.Fatalf("unexpected TarPathsFast error: %v", err)
	}
}

func TestHumanFileSizeHelper(t *testing.T) {
	if got := HumanFileSize(512); got != "512 B" {
		t.Fatalf("unexpected small size: %s", got)
	}
	if got := HumanFileSize(2048); got != "2.0 KB" {
		t.Fatalf("unexpected rounded size: %s", got)
	}
}
