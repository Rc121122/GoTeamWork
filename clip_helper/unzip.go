package clip_helper

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UnzipData unzips the byte slice (data) into a target directory (targetDir).
// It returns the list of extracted file paths (不包含資料夾路徑).
func UnzipData(data []byte, targetDir string) ([]string, error) {
	if len(data) == 0 {
		return nil, errors.New("zip data is empty")
	}

	// 1. 從 byte slice 建立 zip.Reader
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	extractedPaths := make([]string, 0)

	// 2. 建立目標目錄並清理路徑
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	// 清理目標目錄路徑，確保安全檢查的準確性
	targetDir = filepath.Clean(targetDir)

	// 3. 遍歷檔案並解壓縮
	for _, f := range r.File {
		// 構建完整路徑
		fpath := filepath.Join(targetDir, f.Name)

		// !!! 重要的安全檢查: Zip Slip 防禦 !!!
		// 確保解壓路徑位於目標目錄之下，防止惡意 zip 檔案嘗試寫入任意位置。
		// 注意: strings.HasPrefix 必須檢查目標目錄和路徑分隔符號。
		if !strings.HasPrefix(fpath, targetDir+string(os.PathSeparator)) {
			// 檢查是否就是目標目錄本身 (例如 targetDir/. ..)
			if fpath != targetDir {
				return nil, fmt.Errorf("illegal file path in zip (Zip Slip attempted): %s", fpath)
			}
		}

		if f.FileInfo().IsDir() {
			// 處理資料夾
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return nil, err
			}
			continue
		}

		// 處理檔案：確保父資料夾存在
		if err = os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return nil, err
		}

		// 開啟輸出檔案 (O_TRUNC 確保覆蓋現有內容)
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return nil, err
		}

		// 開啟 zip 檔案內的內容
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return nil, err
		}

		// 複製內容
		_, err = io.Copy(outFile, rc)

		// 關閉資源
		rc.Close()
		outFile.Close()

		if err != nil {
			return nil, err
		}

		extractedPaths = append(extractedPaths, fpath)
	}

	return extractedPaths, nil
}
