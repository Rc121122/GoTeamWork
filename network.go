package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"GOproject/clip_helper"
)

const defaultHTTPTimeout = 15 * time.Second

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
			Timeout: 0, // rely on per-request contexts for timeouts
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
		req, err := http.NewRequest("GET", n.serverURL+"/api/users", nil)
		if err != nil {
			lastErr = err
			break
		}
		ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
		req = req.WithContext(ctx)
		resp, err := n.httpClient.Do(req)
		cancel()
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

	req, err := http.NewRequest("POST", n.serverURL+"/api/users", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		n.setDisconnected()
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	defer resp.Body.Close()

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

	req, err := http.NewRequest("POST", n.serverURL+"/api/invite", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := n.httpClient.Do(req)
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

// UploadClipboardItem uploads a clipboard item to the server

func (n *NetworkClient) UploadClipboardItem(item *clip_helper.ClipboardItem, userID, userName string) (*Operation, error) {
	payload := ClipboardUploadRequest{
		Item:     *item,
		UserID:   userID,
		UserName: userName,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", n.serverURL+"/api/clipboard", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()
	httpReq = httpReq.WithContext(ctx)

	resp, err := n.httpClient.Do(httpReq)
	if err != nil {
		n.setDisconnected()
		return nil, fmt.Errorf("failed to upload clipboard item: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var op Operation
	if err := json.NewDecoder(resp.Body).Decode(&op); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &op, nil
}

// UploadZipData uploads archive data for a specific operation (tar format)
func (n *NetworkClient) UploadZipData(opID string, zipData []byte) error {
	return n.uploadFileData(opID, bytes.NewReader(zipData), false, nil)
}

// UploadZipFile uploads archive file from disk for a specific operation
func (n *NetworkClient) UploadZipFile(opID, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer file.Close()
	return n.uploadFileData(opID, file, false, nil)
}

// UploadSingleFileData uploads a raw file instead of a zip for single-file shares
func (n *NetworkClient) UploadSingleFileData(opID string, fileData []byte, name, mime string, size int64, thumb string) error {
	meta := map[string]string{
		"X-Clipboard-File-Name": name,
		"X-Clipboard-File-Mime": mime,
		"X-Clipboard-File-Size": strconv.FormatInt(size, 10),
	}
	if thumb != "" {
		meta["X-Clipboard-File-Thumb"] = thumb
	}
	return n.uploadFileData(opID, bytes.NewReader(fileData), true, meta)
}

// UploadSingleFile uploads a single file from disk
func (n *NetworkClient) UploadSingleFile(opID, filePath, name, mime string, size int64, thumb string) error {
	meta := map[string]string{
		"X-Clipboard-File-Name": name,
		"X-Clipboard-File-Mime": mime,
		"X-Clipboard-File-Size": strconv.FormatInt(size, 10),
	}
	if thumb != "" {
		meta["X-Clipboard-File-Thumb"] = thumb
	}
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	return n.uploadFileData(opID, file, true, meta)
}

func (n *NetworkClient) uploadFileData(opID string, data io.Reader, single bool, headers map[string]string) error {
	endpoint := fmt.Sprintf("%s/api/clipboard/%s/zip", n.serverURL, opID)
	if single {
		endpoint += "?single=1"
	}

	req, err := http.NewRequest("POST", endpoint, data)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if single {
		if headers == nil {
			headers = make(map[string]string)
		}
		contentType := headers["X-Clipboard-File-Mime"]
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		req.Header.Set("Content-Type", contentType)
	} else {
		req.Header.Set("Content-Type", "application/x-tar")
	}

	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		n.setDisconnected()
		return fmt.Errorf("failed to upload file data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}

// setDisconnected marks the client as disconnected
func (n *NetworkClient) setDisconnected() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.connected = false
}

// Ping checks if the server is reachable
func (n *NetworkClient) Ping() error {
	req, err := http.NewRequest("GET", n.serverURL+"/api/users", nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := n.httpClient.Do(req)
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
