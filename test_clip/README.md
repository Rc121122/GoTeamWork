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

## Features Tested

- Clipboard initialization
- Text content monitoring
- Image content monitoring (where supported)
- **File path detection** - automatically identifies when files/folders are copied
- **Native Windows API integration** - uses CF_HDROP format for file detection
- **Smart duplicate prevention** - tracks displayed files to avoid repeated notifications
- **Hybrid detection** - combines event-driven and periodic checking for reliability
- **Clean monitoring** - reads clipboard without modifying content
- Cross-platform compatibility
- Permission handling

## Dependencies

- `golang.design/x/clipboard` - Cross-platform clipboard access
- Platform-specific system libraries (automatically handled by CGO)