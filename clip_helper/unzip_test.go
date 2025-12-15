package clip_helper_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "GOproject/clip_helper" // 匯入 clip_helper 套件
)

// TStamp 用於測試中統一 zip 檔案的時間戳
var TStamp = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// setupZipData 建立一個測試用的 zip 檔案數據
func setupZipData(t *testing.T) []byte {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// 1. 建立一個根檔案
	addFileToZip(t, zipWriter, "testfile1.txt", "Hello, World 1")

	// 2. 建立一個巢狀資料夾和檔案
	addFileToZip(t, zipWriter, "nested/folder/testfile2.log", "Log entry 2")

	// 3. 建立一個空資料夾
	header := &zip.FileHeader{
		Name:     "empty_folder/",
		Method:   zip.Deflate,
		Modified: TStamp,
	}
	if _, err := zipWriter.CreateHeader(header); err != nil {
		t.Fatalf("Failed to create empty folder header: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	return buf.Bytes()
}

// addFileToZip 是一個輔助函式，用於將內容寫入 zip writer
func addFileToZip(t *testing.T, zipWriter *zip.Writer, pathInZip string, content string) {
	header := &zip.FileHeader{
		Name:     pathInZip,
		Method:   zip.Deflate,
		Modified: TStamp,
	}
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		t.Fatalf("Failed to create file header %s: %v", pathInZip, err)
	}
	_, err = writer.Write([]byte(content))
	if err != nil {
		t.Fatalf("Failed to write content to zip %s: %v", pathInZip, err)
	}
}

// testZipSlip 檢查 UnzipData 的安全防護是否有效
func testZipSlip(t *testing.T, baseDir string) {
	t.Helper()

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// 嘗試寫入一個惡意的路徑，它會嘗試跳出 baseDir
	maliciousPath := filepath.Join("..", "malicious_file.txt")
	header := &zip.FileHeader{
		Name:     maliciousPath,
		Method:   zip.Deflate,
		Modified: TStamp,
	}
	if _, err := zipWriter.CreateHeader(header); err != nil {
		t.Fatalf("Failed to create malicious header: %v", err)
	}

	zipWriter.Close()

	_, err := UnzipData(buf.Bytes(), baseDir)

	// 我們期望 UnzipData 會因為安全檢查失敗而回傳錯誤
	if err == nil {
		t.Errorf("Expected UnzipData to fail due to Zip Slip attack, but it succeeded")
	} else if !strings.Contains(err.Error(), "illegal file path in zip") {
		t.Errorf("UnzipData failed, but not with the expected Zip Slip error. Got: %v", err)
	}

	// 額外檢查：確認外部檔案並未真的被創建 (修復 'info' 未使用錯誤)
	externalPath := filepath.Join(filepath.Dir(baseDir), "malicious_file.txt")
	if _, statErr := os.Stat(externalPath); statErr == nil {
		os.Remove(externalPath) // 如果意外創建了，嘗試刪除
		t.Errorf("Zip Slip attack succeeded: external file was created at %s", externalPath)
	}
}

// TestUnzipData 是實際的單元測試函式
func TestUnzipData(t *testing.T) {
	// 1. 準備測試數據
	zipData := setupZipData(t)
	if len(zipData) == 0 {
		t.Fatal("Setup failed: zip data is empty")
	}

	// 2. 建立一個臨時目錄作為解壓縮目標
	tempDir, err := os.MkdirTemp("", "unzip_test_output_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 3. 執行測試函式 (正常流程)
	extractedPaths, err := UnzipData(zipData, tempDir)
	if err != nil {
		t.Fatalf("UnzipData failed unexpectedly: %v", err)
	}

	// 4. 驗證結果
	expectedFilePaths := []string{
		filepath.Join(tempDir, "testfile1.txt"),
		filepath.Join(tempDir, "nested", "folder", "testfile2.log"),
	}

	// 檢查回傳路徑數量 (應只包含檔案路徑)
	if len(extractedPaths) != 2 {
		t.Errorf("Expected 2 extracted file paths, got %d. Paths: %v", len(extractedPaths), extractedPaths)
	}

	// 檢查實際解壓縮的檔案和內容
	for _, expectedPath := range expectedFilePaths {
		// 檢查檔案是否存在
		if _, err := os.Stat(expectedPath); err != nil {
			t.Fatalf("Expected file not found: %s", expectedPath)
		}

		// 檢查檔案內容
		content, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("Failed to read file %s: %v", expectedPath, err)
		}

		if strings.Contains(expectedPath, "testfile1.txt") {
			if string(content) != "Hello, World 1" {
				t.Errorf("File %s content mismatch. Got: %s", expectedPath, string(content))
			}
		} else if strings.Contains(expectedPath, "testfile2.log") {
			if string(content) != "Log entry 2" {
				t.Errorf("File %s content mismatch. Got: %s", expectedPath, string(content))
			}
		}
	}

	// 5. 執行安全測試 (Zip Slip)
	t.Run("ZipSlip", func(t *testing.T) {
		testZipSlip(t, tempDir)
	})

	// 6. 測試空資料
	t.Run("EmptyData", func(t *testing.T) {
		_, err := UnzipData([]byte{}, tempDir)
		if err == nil || !strings.Contains(err.Error(), "zip data is empty") {
			t.Errorf("Expected 'zip data is empty' error, got: %v", err)
		}
	})
}
