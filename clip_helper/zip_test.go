package clip_helper

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestZipLargeFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "zip_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a large file (e.g., 10MB)
	largeFileName := filepath.Join(tmpDir, "large_file.bin")
	f, err := os.Create(largeFileName)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Write 10MB of random data
	// Using a smaller size for CI/CD speed if needed, but 10MB is fine for local
	size := 10 * 1024 * 1024 // 10MB
	buf := make([]byte, 1024*1024) // 1MB buffer
	for i := 0; i < size/(1024*1024); i++ {
		_, err := rand.Read(buf)
		if err != nil {
			f.Close()
			t.Fatalf("Failed to generate random data: %v", err)
		}
		if _, err := f.Write(buf); err != nil {
			f.Close()
			t.Fatalf("Failed to write to large file: %v", err)
		}
	}
	f.Close()

	// Test zipping
	var zipBuf bytes.Buffer
	err = ZipFiles([]string{largeFileName}, &zipBuf)
	if err != nil {
		t.Fatalf("Failed to zip large file: %v", err)
	}

	if zipBuf.Len() == 0 {
		t.Errorf("Zip output is empty")
	}

	t.Logf("Successfully zipped 10MB file. Zip size: %d bytes", zipBuf.Len())
}
