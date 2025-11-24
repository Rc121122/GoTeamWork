//go:build !darwin

package main

func hasAccessibilityPermission() bool {
	return true
}

func requestAccessibilityPermission() bool {
	return true
}
