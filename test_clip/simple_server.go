package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("Starting simple HTTP server test...")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Received request: %s %s\n", r.Method, r.URL.Path)
		fmt.Fprintf(w, "Hello from simple server!")
	})

	fmt.Println("Server starting on :3000")
	fmt.Printf("Attempting to listen on :3000...\n")
	err := http.ListenAndServe(":3000", nil)
	fmt.Printf("ListenAndServe returned: %v\n", err)
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	} else {
		fmt.Println("Server stopped normally")
	}
}
