package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// Configuration constants
const (
	// The port number to scan on each host
	TargetPort = "80"
	// Timeout for attempting a connection to each IP
	Timeout = 300 * time.Millisecond
)

// ScanHost attempts to connect to the specified IP and port
func ScanHost(ip string, port string, wg *sync.WaitGroup, results chan string) {
	// Notify the WaitGroup when the scanning is done
	defer wg.Done()

	target := fmt.Sprintf("%s:%s", ip, port)

	// net.DialTimeout attempts to establish a TCP connection with a set timeout
	conn, err := net.DialTimeout("tcp", target, Timeout)
	if err == nil {
		conn.Close()
		// Send the discovered result to the channel
		results <- fmt.Sprintf("FOUND: %s is up (Port %s is open)", ip, port)
	}
}

// GetSubnetBaseIP retrieves the base IP address of the local subnet (e.g., 192.168.1.)
func GetSubnetBaseIP() (string, error) {
	// Get all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, i := range interfaces {
		// Skip loopback interfaces and those that are not up
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Get the interface's addresses
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			// Check if it is an IP network address
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				// Ensure it is an IPv4 address
				if ipNet.IP.To4() != nil {
					// Convert the IPv4 address to 4 bytes
					ip := ipNet.IP.To4()

					// Extract the first three octets as the subnet base (e.g., 192.168.1)
					// Append a "." for subsequent iteration
					return fmt.Sprintf("%d.%d.%d.", ip[0], ip[1], ip[2]), nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not determine local subnet base IP")
}

func main() {
	fmt.Println(" Starting LAN Scanner...")

	subnetBase, err := GetSubnetBaseIP()
	if err != nil {
		fmt.Printf("ERROR: Failed to get local subnet IP. Please ensure you are connected to a network. Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scanning subnet range: %s1 to %s254 (Port %s)\n", subnetBase, subnetBase, TargetPort)

	// WaitGroup is used to wait for all Goroutines to finish
	var wg sync.WaitGroup
	// Channel is used to receive scan results
	results := make(chan string)

	startTime := time.Now()
	scanCount := 0

	// Iterate through all possible IP addresses (from x.x.x.1 to x.x.x.254)
	for i := 1; i <= 254; i++ {
		ip := fmt.Sprintf("%s%d", subnetBase, i)
		wg.Add(1)
		scanCount++
		// Launch a Goroutine to perform concurrent scanning
		go ScanHost(ip, TargetPort, &wg, results)
	}

	// Start a Goroutine to listen for results from the channel
	go func() {
		for result := range results {
			fmt.Println(result)
		}
	}()

	// Wait for all scanning Goroutines to complete
	wg.Wait()

	// Close the results channel
	close(results)

	elapsed := time.Since(startTime)
	fmt.Printf("\nScan complete! Scanned %d IPs in %s\n", scanCount, elapsed)
}
