# Clipboard Testing Programs

This directory contains platform-specific clipboard testing programs for the GoTeamWork project.

## Files

### macOS Testing
- `clipmacos.go` - Clipboard monitoring test for macOS
- `pasteboard_darwin.go` - macOS pasteboard utilities

### Windows Testing
- `clipwindows.go` - Clipboard monitoring test for Windows

## Building

### macOS
```bash
go build -tags darwin -o clipmacos clipmacos.go pasteboard_darwin.go
```

### Windows
```bash
go build -tags windows -o clipwindows.exe clipwindows.go
```

## Running

### macOS
```bash
./clipmacos
```
- Monitors clipboard changes
- Press `Cmd+Shift+C` to copy content
- Shows file paths and text content

### Windows
```bash
./clipwindows.exe
```
- Monitors clipboard changes
- Press `Ctrl+C` to copy content
- Shows text and image content
- **Automatically detects file paths** when copying files or folders
- **Uses Windows API** to detect file copies (CF_HDROP format)
- **Smart duplicate prevention** - tracks displayed files to avoid repeated notifications

### Windows Test Mode
```bash
./clipwindows.exe --test
```
- Runs comprehensive tests for all features
- Creates test files and ZIP archives
- Validates ZIP compression functionality
- Shows what happens during file copying scenarios

## Features Tested

- Clipboard initialization
- Text content monitoring
- Image content monitoring (where supported)
- **File path detection** - automatically identifies when files/folders are copied
- **Native Windows API integration** - uses CF_HDROP format for file detection
- **Smart duplicate prevention** - tracks displayed files to avoid repeated notifications
- **ZIP compression** - automatically creates ZIP archives for shared files
- **HTTP file server** - provides download links for compressed archives
- **Multi-file support** - handles single files and multiple file selections
- **Automatic cleanup** - removes temporary ZIP files when done

## File Sharing Workflow

1. **Copy Files**: In Windows Explorer, select and copy files/folders (Ctrl+C)
2. **Automatic Detection**: Program detects file paths via Windows clipboard API
3. **ZIP Creation**: Files are automatically compressed into a ZIP archive in the **current working directory**
4. **Server Start**: HTTP server starts on available port (8080-8089)
5. **Share Link**: Program displays sharing URL (e.g., http://localhost:8080)
6. **Download**: Others can access the link to download the ZIP file
7. **Web Interface**: Shows file list, sizes, and download button
8. **Manual Cleanup**: ZIP files remain in current directory for your reference

## ZIP File Location

**ZIP檔案會在程式運行目錄中創建**，而不是臨時目錄！

- **位置**: 與 `clipwindows.exe` 相同的目錄
- **命名**: 
  - 單一檔案: `share_[時間戳].zip`
  - 多檔案: `shared_files_[時間戳].zip`
- **清理**: 程式結束時ZIP檔案會保留，需要手動刪除

### 範例
```
c:\go\GoTeamWork\test_clip\
├── clipwindows.exe
├── share_1733280000.zip      ← 單一檔案的ZIP
├── shared_files_1733280001.zip ← 多檔案的ZIP
└── README.md
```

## Technical Details

- **ZIP Format**: Uses Go's archive/zip for compression
- **HTTP Server**: Built-in net/http server with custom file listing
- **Windows Integration**: CGO with Windows API for clipboard file detection
- **Memory Efficient**: Streams files directly to ZIP without loading entirely in memory
- **Cross-platform Ready**: Core logic works on all platforms (Windows-specific detection)
- **Hybrid detection** - combines event-driven and periodic checking for reliability
- **Clean monitoring** - reads clipboard without modifying content
- Cross-platform compatibility
- Permission handling

## Dependencies

- `golang.design/x/clipboard` - Cross-platform clipboard access
- Platform-specific system libraries (automatically handled by CGO)