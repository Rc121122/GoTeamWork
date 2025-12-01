//go:build windows
// +build windows

package main

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"golang.design/x/clipboard"
)

/*
#cgo windows CFLAGS: -DWIN32 -DWINVER=0x0600
#cgo windows LDFLAGS: -lole32 -lshell32 -luser32

#include <windows.h>
#include <shlobj.h>
#include <ole2.h>
#include <stdlib.h>

static char** GetClipboardFilePaths() {
    if (!OpenClipboard(NULL)) {
        return NULL;
    }

    HANDLE hData = GetClipboardData(CF_HDROP);
    if (hData == NULL) {
        CloseClipboard();
        return NULL;
    }

    HDROP hDrop = (HDROP)hData;
    UINT fileCount = DragQueryFile(hDrop, 0xFFFFFFFF, NULL, 0);

    if (fileCount == 0) {
        CloseClipboard();
        return NULL;
    }

    char** paths = (char**)malloc(sizeof(char*) * (fileCount + 1));
    if (paths == NULL) {
        CloseClipboard();
        return NULL;
    }

    for (UINT i = 0; i < fileCount; i++) {
        UINT pathLen = DragQueryFile(hDrop, i, NULL, 0) + 1;
        paths[i] = (char*)malloc(pathLen);
        if (paths[i] == NULL) {
            // Clean up previously allocated paths
            for (UINT j = 0; j < i; j++) {
                free(paths[j]);
            }
            free(paths);
            CloseClipboard();
            return NULL;
        }
        DragQueryFile(hDrop, i, paths[i], pathLen);
    }
    paths[fileCount] = NULL;

    CloseClipboard();
    return paths;
}
*/
import "C"

// getFilePathsFromClipboard returns absolute file paths currently stored in the Windows clipboard.
func getFilePathsFromClipboard() []string {
	pathsPtr := C.GetClipboardFilePaths()
	if pathsPtr == nil {
		return nil
	}
	defer func() {
		// Free the C array
		for i := 0; ; i++ {
			pathPtr := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(pathsPtr)) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
			if pathPtr == nil {
				break
			}
			C.free(unsafe.Pointer(pathPtr))
		}
		C.free(unsafe.Pointer(pathsPtr))
	}()

	var paths []string
	for i := 0; ; i++ {
		pathPtr := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(pathsPtr)) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		if pathPtr == nil {
			break
		}
		path := C.GoString(pathPtr)
		paths = append(paths, path)
	}

	return paths
}

func main() {
	fmt.Println("Welcome to clipboard testing for Windows, waiting for response, press \"Ctrl+C\" to copy here.")

	ctx := context.Background()

	// Watch for text changes
	chText := clipboard.Watch(ctx, clipboard.FmtText)

	// Watch for image changes (if supported)
	chImage := clipboard.Watch(ctx, clipboard.FmtImage)

	// Test initial clipboard content
	if err := clipboard.Init(); err != nil {
		fmt.Printf("Failed to initialize clipboard: %v\n", err)
		return
	}

	fmt.Println("Clipboard initialized successfully")

	// Read initial content
	if textData := clipboard.Read(clipboard.FmtText); len(textData) > 0 {
		fmt.Printf("Initial clipboard text: %s\n", string(textData))
	}

	if imgData := clipboard.Read(clipboard.FmtImage); len(imgData) > 0 {
		fmt.Printf("Initial clipboard has image data (%d bytes)\n", len(imgData))
	}

	fmt.Println("Monitoring clipboard changes...")

	for {
		select {
		case data := <-chText:
			fmt.Printf("DEBUG: Text clipboard changed, length: %d\n", len(data))

			// Always check for file paths when any clipboard change occurs
			if filePaths := getFilePathsFromClipboard(); len(filePaths) > 0 {
				for _, path := range filePaths {
					fmt.Printf("You copied file: %s\n", path)
				}
				continue
			}

			text := string(data)

			// Skip empty or whitespace-only text
			if strings.TrimSpace(text) == "" {
				fmt.Printf("DEBUG: Skipping empty text\n")
				continue
			}

			fmt.Printf("You copied text: %s\n", text)

			// Check if it looks like file paths
			if isFilePath(text) {
				fmt.Printf("Detected file path(s): %s\n", text)
			} else if len(strings.TrimSpace(text)) > 0 {
				// Additional check: if text contains file-like patterns but not detected as full paths
				if strings.Contains(text, "\\") || strings.Contains(text, "/") {
					fmt.Printf("Possible file reference: %s\n", text)
				}
			}

			// Only modify clipboard if it's not already modified text (avoid infinite loop)
			if !strings.HasPrefix(text, "Modified: ") {
				// Test writing back to clipboard
				testText := fmt.Sprintf("Modified: %s", text)
				clipboard.Write(clipboard.FmtText, []byte(testText))
				fmt.Printf("Modified clipboard to: %s\n", testText)
			}

		case data := <-chImage:
			fmt.Printf("DEBUG: Image clipboard changed, size: %d bytes\n", len(data))
			if len(data) > 0 {
				fmt.Printf("You copied an image (%d bytes)\n", len(data))

				// Test writing back (though we can't modify image data easily)
				fmt.Println("Image copied to clipboard")
			}

		case <-time.After(30 * time.Second):
			fmt.Println("No clipboard activity for 30 seconds...")
		}
	}
}

// isFilePath checks if the text contains file paths
func isFilePath(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check if it looks like a file path (has extension or directory structure)
		if strings.Contains(line, "\\") || strings.Contains(line, "/") {
			// Additional check for common file extensions or drive letters
			if strings.Contains(line, ".") || strings.HasPrefix(line, "C:") || strings.HasPrefix(line, "D:") {
				return true
			}
		}
	}
	return false
}
