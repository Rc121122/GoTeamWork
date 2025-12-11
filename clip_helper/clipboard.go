package clip_helper

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ClipboardItemType represents the type of clipboard content
type ClipboardItemType string

const (
	ClipboardText  ClipboardItemType = "text"
	ClipboardImage ClipboardItemType = "image"
	ClipboardFile  ClipboardItemType = "file" // New type for files/zip
)

// ClipboardItem represents a clipboard item with its content
type ClipboardItem struct {
	Type    ClipboardItemType `json:"type"`
	Text    string            `json:"text,omitempty"`
	Image   []byte            `json:"image,omitempty"` // PNG encoded
	ZipData []byte            `json:"-"`               // Zip file content
	Files   []string          `json:"files,omitempty"` // File paths
}

const (
	ClipboardShareCooldown = 250 * time.Millisecond
	ClipboardCacheTTL      = 8 * time.Second
)

// ZipFiles creates a zip archive containing the specified files
func ZipFiles(files []string, writer io.Writer) error {
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	for _, file := range files {
		if err := addFileToZip(zipWriter, file); err != nil {
			return err
		}
	}
	return nil
}

// addFileToZip adds a file or directory to the zip archive
func addFileToZip(zipWriter *zip.Writer, filename string) error {
	// Check if file exists
	if _, err := os.Stat(filename); err != nil {
		return err
	}

	baseDir := filepath.Dir(filename)

	return filepath.Walk(filename, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Use forward slashes for zip paths
		header.Name = filepath.ToSlash(relPath)

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
