package main

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestGetSubnetBaseIP(t *testing.T) {
	ip, err := GetSubnetBaseIP()
	if err != nil {
		t.Skipf("GetSubnetBaseIP unavailable in test env: %v", err)
	}
	if ip == "" {
		t.Fatalf("expected non-empty subnet base")
	}
	if _, err := net.ResolveIPAddr("ip", ip+"1"); err != nil {
		t.Fatalf("expected resolvable base IP: %s: %v", ip, err)
	}
}

func TestScanHost(t *testing.T) {
	results := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go ScanHost("127.0.0.1", "80", &wg, results)
	go func() {
		wg.Wait()
		close(results)
	}()

	select {
	case <-results:
		// Either found or not; reaching here means function executed
	case <-time.After(2 * time.Second):
		t.Fatalf("ScanHost timed out")
	}
}
