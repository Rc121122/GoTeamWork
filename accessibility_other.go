//go:build !darwin

package main

import (
	"runtime"

	"golang.design/x/clipboard"
)

// hasAccessibilityPermission checks if the application has permission to access clipboard
func hasAccessibilityPermission() bool {
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
	// On other systems, assume permission is granted
	return true
}

// requestAccessibilityPermission requests clipboard access permission
func requestAccessibilityPermission() bool {
	if runtime.GOOS == "windows" {
		// On Windows, just try to initialize clipboard
		// If it fails, the user might need to run as administrator or check permissions
		err := clipboard.Init()
		return err == nil
	} else if runtime.GOOS == "linux" {
		// On Linux, clipboard access is usually available
		// If initialization fails, it might be due to display server issues
		err := clipboard.Init()
		return err == nil
	}
	// On other systems, assume permission can be requested
	return true
}
