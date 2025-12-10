//go:build !darwin

package clip_helper

import (
	"runtime"

	"golang.design/x/clipboard"
)

// HasAccessibilityPermission checks if the application has permission to access clipboard
func HasAccessibilityPermission() bool {
	if runtime.GOOS == "windows" {
		// On Windows, try to initialize clipboard to check permissions
		err := clipboard.Init()
		return err == nil
	} else if runtime.GOOS == "linux" {
		// On Linux, clipboard access usually works if display server is available
		// Try to initialize clipboard as a basic check
		err := clipboard.Init()
		return err == nil
	}
	return true
}

// RequestAccessibilityPermission requests permission to access clipboard
func RequestAccessibilityPermission() bool {
	// On Windows and Linux, we can't explicitly request permission like macOS
	// But we can try to initialize clipboard again
	return HasAccessibilityPermission()
}
