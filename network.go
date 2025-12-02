package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// NetworkClient handles communication with the central server
type NetworkClient struct {
	serverURL  string
	httpClient *http.Client
	connected  bool
	mu         sync.RWMutex
}

// NewNetworkClient creates a new network client
func NewNetworkClient(serverURL string) *NetworkClient {
	return &NetworkClient{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		connected: false,
	}
}

// ConnectToServer establishes connection to the central server with retry logic
func (n *NetworkClient) ConnectToServer() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	const maxRetries = 3
	const retryDelay = 2 * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Try to ping the server
		resp, err := n.httpClient.Get(n.serverURL + "/api/users")
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				fmt.Printf("Connection attempt %d failed, retrying in %v: %v\n", attempt, retryDelay, err)
				time.Sleep(retryDelay)
				continue
			}
			n.connected = false
			return fmt.Errorf("failed to connect to server after %d attempts: %w", maxRetries, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("server returned status: %d", resp.StatusCode)
			if attempt < maxRetries {
				fmt.Printf("Connection attempt %d failed (status %d), retrying in %v\n", attempt, resp.StatusCode, retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			n.connected = false
			return lastErr
		}

		n.connected = true
		fmt.Println("Successfully connected to central server:", n.serverURL)
		return nil
	}

	n.connected = false
	return lastErr
}

// IsConnected returns the current connection status
func (n *NetworkClient) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.connected
}

// CreateUser creates a new user on the server
func (n *NetworkClient) CreateUser(name string) (*User, error) {
	reqBody := map[string]string{"name": name}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := n.httpClient.Post(
		n.serverURL+"/api/users",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		n.setDisconnected()
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return nil, fmt.Errorf("username already exists")
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}

	return &user, nil
}

// SendInvite sends an invitation to another user with a custom message
func (n *NetworkClient) SendInvite(inviteeID, inviterID, message string) (string, error) {
	reqBody := map[string]string{
		"userId":    inviteeID,
		"inviterId": inviterID,
		"message":   message,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := n.httpClient.Post(
		n.serverURL+"/api/invite",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		n.setDisconnected()
		return "", fmt.Errorf("failed to send invite: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result["message"], nil
}

// setDisconnected marks the client as disconnected
func (n *NetworkClient) setDisconnected() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.connected = false
}

// Ping checks if the server is reachable
func (n *NetworkClient) Ping() error {
	resp, err := n.httpClient.Get(n.serverURL + "/api/users")
	if err != nil {
		n.setDisconnected()
		return fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		n.setDisconnected()
		return fmt.Errorf("server unreachable, status: %d", resp.StatusCode)
	}

	n.mu.Lock()
	n.connected = true
	n.mu.Unlock()

	return nil
}
