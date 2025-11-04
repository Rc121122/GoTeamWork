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

	// Local cache for fetched data
	cachedUsers []*User
	cachedRooms []*Room
	lastSync    time.Time
}

// NewNetworkClient creates a new network client
func NewNetworkClient(serverURL string) *NetworkClient {
	return &NetworkClient{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		connected:   false,
		cachedUsers: make([]*User, 0),
		cachedRooms: make([]*Room, 0),
	}
}

// ConnectToServer establishes connection to the central server
func (n *NetworkClient) ConnectToServer() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Try to ping the server
	resp, err := n.httpClient.Get(n.serverURL + "/api/users")
	if err != nil {
		n.connected = false
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		n.connected = false
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	n.connected = true
	fmt.Println("Successfully connected to central server:", n.serverURL)
	return nil
}

// IsConnected returns the current connection status
func (n *NetworkClient) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.connected
}

// FetchUsers retrieves all users from the server
func (n *NetworkClient) FetchUsers() ([]*User, error) {
	resp, err := n.httpClient.Get(n.serverURL + "/api/users")
	if err != nil {
		n.setDisconnected()
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var users []*User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("failed to parse users: %w", err)
	}

	// Update cache
	n.mu.Lock()
	n.cachedUsers = users
	n.lastSync = time.Now()
	n.mu.Unlock()

	return users, nil
}

// FetchRooms retrieves all rooms from the server
func (n *NetworkClient) FetchRooms() ([]*Room, error) {
	resp, err := n.httpClient.Get(n.serverURL + "/api/rooms")
	if err != nil {
		n.setDisconnected()
		return nil, fmt.Errorf("failed to fetch rooms: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rooms []*Room
	if err := json.Unmarshal(body, &rooms); err != nil {
		return nil, fmt.Errorf("failed to parse rooms: %w", err)
	}

	// Update cache
	n.mu.Lock()
	n.cachedRooms = rooms
	n.lastSync = time.Now()
	n.mu.Unlock()

	return rooms, nil
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

// SendInvite sends an invitation to another user
func (n *NetworkClient) SendInvite(userID string) (string, error) {
	reqBody := map[string]string{"userId": userID}
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

// SyncData periodically fetches data from server and updates local cache
func (n *NetworkClient) SyncData() error {
	// Fetch users
	users, err := n.FetchUsers()
	if err != nil {
		return fmt.Errorf("failed to sync users: %w", err)
	}

	// Fetch rooms
	rooms, err := n.FetchRooms()
	if err != nil {
		return fmt.Errorf("failed to sync rooms: %w", err)
	}

	fmt.Printf("Synced data: %d users, %d rooms\n", len(users), len(rooms))
	return nil
}

// GetCachedUsers returns the locally cached users
func (n *NetworkClient) GetCachedUsers() []*User {
	n.mu.RLock()
	defer n.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]*User, len(n.cachedUsers))
	copy(result, n.cachedUsers)
	return result
}

// GetCachedRooms returns the locally cached rooms
func (n *NetworkClient) GetCachedRooms() []*Room {
	n.mu.RLock()
	defer n.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]*Room, len(n.cachedRooms))
	copy(result, n.cachedRooms)
	return result
}

// StartAutoSync starts automatic data synchronization
func (n *NetworkClient) StartAutoSync(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if n.IsConnected() {
				if err := n.SyncData(); err != nil {
					fmt.Printf("Auto-sync error: %v\n", err)
				}
			}
		}
	}()
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

// GetLastSyncTime returns the timestamp of the last successful sync
func (n *NetworkClient) GetLastSyncTime() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.lastSync
}
