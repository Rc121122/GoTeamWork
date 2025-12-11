//go:build windows
// +build windows

package clip_helper

import (
	"fmt"
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

// getFilePathsFromClipboard reads file paths from Windows clipboard using CF_HDROP
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

// ReadClipboard reads the current clipboard content and returns a ClipboardItem
func ReadClipboard() (*ClipboardItem, error) {
	// Initialize clipboard if needed
	err := clipboard.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize clipboard: %w", err)
	}

	// Try to read as file paths from Windows clipboard
	if filePaths := getFilePathsFromClipboard(); len(filePaths) > 0 {
		return &ClipboardItem{
			Type:  ClipboardFile,
			Files: filePaths,
		}, nil
	}

	// Try to read as image
	if imgData := clipboard.Read(clipboard.FmtImage); len(imgData) > 0 {
		return &ClipboardItem{
			Type:  ClipboardImage,
			Image: imgData,
		}, nil
	}

	// Try to read as text
	if textData := clipboard.Read(clipboard.FmtText); len(textData) > 0 {
		text := string(textData)
		return &ClipboardItem{
			Type: ClipboardText,
			Text: text,
		}, nil
	}

	return nil, fmt.Errorf("no supported clipboard content found")
}
