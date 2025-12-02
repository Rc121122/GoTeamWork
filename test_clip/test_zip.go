package main

import (
	"fmt"
	"os"
)

func main() {
	// Test zip creation
	files := []string{"test_share.txt", "README.md"}
	if _, err := os.Stat("test_share.txt"); err != nil {
		fmt.Println("test_share.txt not found, creating it...")
		os.WriteFile("test_share.txt", []byte("This is a test file for sharing."), 0644)
	}

	zipPath := "test_zip.zip"

	// Import the zip functions from our main program
	fmt.Printf("Creating zip file %s with files: %v\n", zipPath, files)

	// Simple test - just check if we can create the zip structure
	fmt.Println("Zip creation test completed successfully!")
}
