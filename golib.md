# Go Libraries for System Clipboard Access

This document lists Go packages required for accessing the system clipboard, supporting text, images, and files (file paths). The libraries are cross-platform where possible, with specific notes for Windows and macOS.

## Recommended Library: golang.design/x/clipboard

**Package:** `golang.design/x/clipboard`

**Description:** A cross-platform Go library specifically designed for clipboard operations. It provides comprehensive support for text and images, with a focus on being pure Go where possible (using CGO only when necessary).

**Supported Features:**
- Text: Read and write text content
- Image: Read and write image data
- Files: Not directly supported (file paths can be handled as text if copied as paths)

**Platform Support:**
- **Windows:** Fully supported
- **macOS:** Fully supported
- **Linux:** Fully supported
- **Others:** BSD variants, etc.

**Cross-Platform Usage:** You can use the same package and API across Windows and macOS without needing different packages. The library handles OS-specific clipboard APIs internally.

**Installation:**
```bash
go get golang.design/x/clipboard
```

**Usage Examples:**
```go
import "golang.design/x/clipboard"

// Initialize (required)
clipboard.Init()

// Text clipboard
clipboard.Write(clipboard.FmtText, []byte("Hello, World!"))
text := string(clipboard.Read(clipboard.FmtText))

// Image clipboard
clipboard.Write(clipboard.FmtImage, imageBytes)
imgData := clipboard.Read(clipboard.FmtImage)
```

**Dependencies:**
- May use CGO on some platforms for full functionality
- Pure Go implementation where possible

## Alternative Libraries

### github.com/go-vgo/robotgo
- **Pros:** Supports text, images, and file paths; part of a larger GUI automation toolkit
- **Cons:** Heavier dependency; primarily for automation rather than just clipboard; hotkey detection API issues in current implementation
- **Cross-Platform:** Same package works on Windows and macOS
- **Recommendation:** Used for GUI automation, but golang.design/x/clipboard is better for clipboard-specific operations

### github.com/atotto/clipboard
- **Pros:** Simple, lightweight for text-only operations
- **Cons:** Text only; no image or file support
- **Cross-Platform:** Same package works on Windows and macOS
- **Recommendation:** Not sufficient if you need images or files

## Notes
- **Cross-Platform Compatibility:** golang.design/x/clipboard is the best choice for your requirements as it supports text and images with the same package on both Windows and macOS. File path support can be added by treating paths as text.
- For file access after retrieving paths from clipboard, use standard Go packages like `os` and `io`.
- Hotkey detection for clipboard copy (Ctrl+Shift+C / Cmd+Shift+C) is implemented as a placeholder; may require additional libraries for reliable cross-platform global hotkeys.

