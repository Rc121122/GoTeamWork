import './style.css';
import './app.css';

import { GetMode } from '../wailsjs/go/main/App';

// Global state
let currentMode = 'client';
let currentUser = null;
let currentRoom = null;
let userListInterval = null;
let inviteCheckInterval = null;
let chatHistoryInterval = null;
let isProcessingAction = false; // Prevent rapid actions

// Cleanup function to clear all intervals and event listeners
function cleanup() {
    if (userListInterval) {
        clearInterval(userListInterval);
        userListInterval = null;
    }
    if (inviteCheckInterval) {
        clearInterval(inviteCheckInterval);
        inviteCheckInterval = null;
    }
    if (chatHistoryInterval) {
        clearInterval(chatHistoryInterval);
        chatHistoryInterval = null;
    }
    isProcessingAction = false;
}

// Initialize the app
async function initApp() {
    try {
        currentMode = await GetMode();
        console.log('App mode:', currentMode);

        if (currentMode === 'host') {
            renderHostLobby();
        } else {
            renderClientUI();
        }
    } catch (error) {
        console.error('Failed to get mode:', error);
        // Default to client mode
        renderClientUI();
    }
}

// Render host lobby (shows all users and rooms)
function renderHostLobby() {
    // Cleanup previous intervals
    cleanup();

    document.querySelector('#app').innerHTML = `
        <div class="host-lobby">
            <div class="lobby-header">
                <h1>Server Management Lobby</h1>
                <p>Monitor users and rooms</p>
            </div>

            <div class="lobby-content">
                <div class="users-section">
                    <h2>All Users</h2>
                    <div id="all-users-list" class="users-list">
                        <!-- Users will be loaded here -->
                    </div>
                </div>

                <div class="rooms-section">
                    <h2>All Rooms</h2>
                    <div id="all-rooms-list" class="rooms-list">
                        <!-- Rooms will be loaded here -->
                    </div>
                </div>
            </div>
        </div>
    `;

    // Start updating user and room lists (reduced frequency)
    updateHostLobby();
    setInterval(updateHostLobby, 5000); // Changed from 3000 to 5000ms
}

// Update host lobby with current users and rooms
async function updateHostLobby() {
    try {
        // Update users list
        const usersResponse = await fetch('http://localhost:8080/api/users');
        if (usersResponse.ok) {
            const users = await usersResponse.json();
            renderHostUsersList(users);
        }

        // Update rooms list
        const roomsResponse = await fetch('http://localhost:8080/api/rooms');
        if (roomsResponse.ok) {
            const rooms = await roomsResponse.json();
            renderHostRoomsList(rooms);
        }
    } catch (error) {
        console.error('Failed to update host lobby:', error);
    }
}

// Render users list for host
function renderHostUsersList(users) {
    const usersListDiv = document.getElementById('all-users-list');
    usersListDiv.innerHTML = '';

    if (users.length === 0) {
        usersListDiv.innerHTML = '<p class="empty">No users connected</p>';
        return;
    }

    users.forEach(user => {
        const userDiv = document.createElement('div');
        userDiv.className = 'user-item';

        const roomStatus = user.roomId ? `In room: ${user.roomId}` : 'Not in room';

        userDiv.innerHTML = `
            <div class="user-info">
                <span class="user-name">${user.name}</span>
                <span class="user-status ${user.isOnline ? 'online' : 'offline'}">
                    ● ${user.isOnline ? 'Online' : 'Offline'}
                </span>
                <span class="user-room">${roomStatus}</span>
            </div>
        `;

        usersListDiv.appendChild(userDiv);
    });
}

// Render rooms list for host
function renderHostRoomsList(rooms) {
    const roomsListDiv = document.getElementById('all-rooms-list');
    roomsListDiv.innerHTML = '';

    if (rooms.length === 0) {
        roomsListDiv.innerHTML = '<p class="empty">No active rooms</p>';
        return;
    }

    rooms.forEach(room => {
        const roomDiv = document.createElement('div');
        roomDiv.className = 'room-item';

        roomDiv.innerHTML = `
            <div class="room-info">
                <span class="room-name">${room.name}</span>
                <span class="room-users">Users: ${room.userIds.length}</span>
                <div class="room-user-list">
                    ${room.userIds.join(', ')}
                </div>
            </div>
        `;

        roomsListDiv.appendChild(roomDiv);
    });
}

// Render client UI (username input -> waiting lobby)
function renderClientUI() {
    // Start with username input
    renderUsernameInput();
}

// Render username input screen
function renderUsernameInput() {
    document.querySelector('#app').innerHTML = `
        <div class="username-screen">
            <div class="username-container">
                <h1>GoTeamWork</h1>
                <p>Enter your username to join the collaboration</p>
                <div class="username-input-group">
                    <input type="text" id="username-input" placeholder="Enter username..." maxlength="20" />
                    <button id="join-button">Join</button>
                </div>
                <div id="username-error" class="error-message"></div>
            </div>
        </div>
    `;

    // Add event listeners
    const input = document.getElementById('username-input');
    const button = document.getElementById('join-button');
    const errorDiv = document.getElementById('username-error');

    input.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            joinWithUsername();
        }
    });

    button.addEventListener('click', joinWithUsername);

    async function joinWithUsername() {
        const username = input.value.trim();
        if (!username) {
            showError('Please enter a username');
            return;
        }

        try {
            // Try to create user via HTTP API
            const response = await fetch('http://localhost:8080/api/users', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ name: username }),
            });

            if (response.ok) {
                const user = await response.json();
                currentUser = user;
                renderWaitingLobby();
            } else if (response.status === 409) {
                showError('Username already exists. Please choose another one.');
            } else {
                showError('Failed to join. Please try again.');
            }
        } catch (error) {
            console.error('Error joining:', error);
            showError('Cannot connect to server. Please check if host is running.');
        }
    }

    function showError(message) {
        errorDiv.textContent = message;
        errorDiv.style.display = 'block';
        setTimeout(() => {
            errorDiv.style.display = 'none';
        }, 5000);
    }
}

// Render waiting lobby
function renderWaitingLobby() {
    // Cleanup previous intervals
    cleanup();

    document.querySelector('#app').innerHTML = `
        <div class="waiting-lobby">
            <div class="lobby-header">
                <h1>Waiting Lobby</h1>
                <p>Welcome, ${currentUser.name}! Waiting for others to join...</p>
            </div>
            <div class="user-list-container">
                <h2>Online Users</h2>
                <div id="user-list" class="user-list">
                    <!-- Users will be loaded here -->
                </div>
            </div>
        </div>

        <!-- Invite Modal -->
        <div id="invite-modal" class="modal" style="display: none;">
            <div class="modal-content">
                <h2>Room Invitation</h2>
                <p id="invite-message">You have been invited to join a room!</p>
                <div class="modal-buttons">
                    <button id="accept-invite" class="accept-btn">Accept</button>
                    <button id="decline-invite" class="decline-btn">Decline</button>
                </div>
            </div>
        </div>
    `;

    // Start polling for user updates (reduced frequency)
    updateUserList();
    userListInterval = setInterval(updateUserList, 10000); // Changed from 5000 to 10000ms

    // Start polling for invites (reduced frequency)
    checkForInvites();
    inviteCheckInterval = setInterval(checkForInvites, 5000); // Changed from 2000 to 5000ms
}

// Update user list from server
async function updateUserList() {
    try {
        const response = await fetch('http://localhost:8080/api/users');
        if (response.ok) {
            const users = await response.json();
            renderUserList(users);
        } else {
            console.warn('Failed to fetch users, status:', response.status);
        }
    } catch (error) {
        console.error('Failed to fetch users:', error);
        // Don't show error to user, just log it
    }
}

// Render user list in waiting lobby
function renderUserList(users) {
    const userListDiv = document.getElementById('user-list');
    if (!userListDiv) {
        console.warn('user-list element not found');
        return;
    }

    userListDiv.innerHTML = '';

    if (!Array.isArray(users)) {
        console.warn('Users data is not an array:', users);
        userListDiv.innerHTML = '<p class="empty">Error loading users</p>';
        return;
    }

    if (users.length === 0) {
        userListDiv.innerHTML = '<p class="empty">No other users online</p>';
        return;
    }

    // Filter out current user and show other users
    const otherUsers = users.filter(user => user && user.id !== currentUser?.id);

    if (otherUsers.length === 0) {
        userListDiv.innerHTML = '<p class="empty">No other users online</p>';
        return;
    }

    otherUsers.forEach(user => {
        if (!user || !user.id || !user.name) {
            console.warn('Invalid user data:', user);
            return;
        }

        const userDiv = document.createElement('div');
        userDiv.className = 'user-item';

        const roomStatus = user.roomId ? `In room: ${user.roomId}` : 'Waiting in lobby';

        userDiv.innerHTML = `
            <div class="user-info">
                <span class="user-name">${user.name}</span>
                <span class="user-status ${user.isOnline ? 'online' : 'offline'}">
                    ● ${user.isOnline ? 'Online' : 'Offline'}
                </span>
                <span class="user-room">${roomStatus}</span>
            </div>
        `;

        userListDiv.appendChild(userDiv);
    });
}

// Check for pending invites
async function checkForInvites() {
    if (!currentUser) return;

    try {
        // Check if user is now in a room
        const userResponse = await fetch(`http://localhost:8080/api/users/${currentUser.id}`);
        if (userResponse.ok) {
            const updatedUser = await userResponse.json();

            // If user is now in a room but wasn't before, show invite modal
            if (updatedUser.roomId && (!currentUser.roomId || currentUser.roomId !== updatedUser.roomId)) {
                currentUser = updatedUser;
                showInviteModal();
            }
        } else {
            console.warn('Failed to check for invites, status:', userResponse.status);
        }
    } catch (error) {
        console.error('Failed to check for invites:', error);
        // Don't show error to user, just log it
    }
}

// Show invite modal
function showInviteModal() {
    const modal = document.getElementById('invite-modal');
    const acceptBtn = document.getElementById('accept-invite');
    const declineBtn = document.getElementById('decline-invite');

    modal.style.display = 'flex';

    acceptBtn.onclick = () => {
        modal.style.display = 'none';
        renderRoomView();
    };

    declineBtn.onclick = () => {
        modal.style.display = 'none';
        // Optionally leave the room if declined
        leaveRoom();
    };
}

// Leave current room
async function leaveRoom() {
    if (!currentUser || !currentUser.roomId) return;

    try {
        // Note: This would need a LeaveRoom API endpoint
        // For now, just update local state
        currentUser.roomId = null;
        renderWaitingLobby();
    } catch (error) {
        console.error('Failed to leave room:', error);
    }
}

// Render room view (chat + clipboard)
function renderRoomView() {
    if (!currentUser || !currentUser.roomId) {
        renderWaitingLobby();
        return;
    }

    // Cleanup previous intervals
    cleanup();

    document.querySelector('#app').innerHTML = `
        <div class="room-view">
            <div class="room-header">
                <h1>Room: ${currentUser.roomId}</h1>
                <button id="leave-room-btn" class="leave-btn">Leave Room</button>
            </div>

            <div class="room-content">
                <div class="chat-section">
                    <h2>Chat</h2>
                    <div id="room-chat-messages" class="chat-messages">
                        <!-- Chat messages will be loaded here -->
                    </div>
                    <div class="chat-input-area">
                        <input type="text" id="room-message-input" placeholder="Type a message..." />
                        <button id="room-send-button">Send</button>
                    </div>
                </div>

                <div class="clipboard-section">
                    <h2>Shared Clipboard</h2>
                    <div id="room-clipboard-content" class="clipboard-content">
                        <!-- Shared content will be displayed here -->
                    </div>
                    <div class="clipboard-input-area">
                        <textarea id="room-clipboard-text" placeholder="Paste content to share..."></textarea>
                        <button id="room-share-button">Share</button>
                    </div>
                </div>
            </div>
        </div>
    `;

    // Load chat history
    loadRoomChatHistory();

    // Add event listeners
    document.getElementById('leave-room-btn').addEventListener('click', () => {
        leaveRoom();
    });

    document.getElementById('room-send-button').addEventListener('click', () => {
        const input = document.getElementById('room-message-input');
        const message = input.value.trim();
        if (message) {
            sendRoomMessage(message);
            input.value = '';
        }
    });

    document.getElementById('room-share-button').addEventListener('click', () => {
        const textarea = document.getElementById('room-clipboard-text');
        const content = textarea.value.trim();
        if (content) {
            shareRoomClipboard(content);
            textarea.value = '';
        }
    });

    // Start polling for new messages (reduced frequency)
    chatHistoryInterval = setInterval(loadRoomChatHistory, 5000); // Changed from 2000 to 5000ms
}

// Invite a user
async function inviteUser(userId, userName) {
    try {
        const response = await fetch('http://localhost:8080/api/invite', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ userId: userId }),
        });

        if (response.ok) {
            const result = await response.json();
            alert(result.message);
            // Could transition to room view here
        } else {
            alert('Failed to send invitation');
        }
    } catch (error) {
        console.error('Error inviting user:', error);
        alert('Cannot connect to server');
    }
}

// Send message to room
async function sendRoomMessage(message) {
    if (!currentUser || !currentUser.roomId || isProcessingAction) return;

    isProcessingAction = true;

    try {
        const response = await fetch('http://localhost:8080/api/chat', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                roomId: currentUser.roomId,
                userId: currentUser.id,
                message: message,
            }),
        });

        if (response.ok) {
            loadRoomChatHistory(); // Refresh messages
        } else {
            console.error('Failed to send message, status:', response.status);
        }
    } catch (error) {
        console.error('Error sending message:', error);
    } finally {
        isProcessingAction = false;
    }
}

// Load chat history for current room
async function loadRoomChatHistory() {
    if (!currentUser || !currentUser.roomId) return;

    try {
        const response = await fetch(`http://localhost:8080/api/chat/${currentUser.roomId}`);
        if (response.ok) {
            const messages = await response.json();
            renderRoomChatMessages(messages);
        } else {
            console.warn('Failed to load chat history, status:', response.status);
        }
    } catch (error) {
        console.error('Failed to load chat history:', error);
        // Don't show error to user, just log it
    }
}

// Render chat messages in room
function renderRoomChatMessages(messages) {
    const chatDiv = document.getElementById('room-chat-messages');
    if (!chatDiv) {
        console.warn('room-chat-messages element not found');
        return;
    }

    if (!Array.isArray(messages)) {
        console.warn('Messages data is not an array:', messages);
        return;
    }

    chatDiv.innerHTML = '';

    messages.forEach(message => {
        if (!message || !message.userName || !message.message) {
            console.warn('Invalid message data:', message);
            return;
        }

        const messageDiv = document.createElement('div');
        messageDiv.className = 'chat-message';

        const timestamp = message.timestamp ? new Date(message.timestamp * 1000).toLocaleTimeString() : new Date().toLocaleTimeString();

        messageDiv.innerHTML = `
            <div class="message-header">
                <span class="message-user">${message.userName}</span>
                <span class="message-time">${timestamp}</span>
            </div>
            <div class="message-content">${message.message}</div>
        `;

        chatDiv.appendChild(messageDiv);
    });

    // Scroll to bottom
    chatDiv.scrollTop = chatDiv.scrollHeight;
}

// Share clipboard content in room
function shareRoomClipboard(content) {
    if (isProcessingAction) return;

    isProcessingAction = true;

    // For now, store in localStorage with room prefix
    // In a real implementation, this would be sent to the server
    const key = `room_${currentUser.roomId}_clipboard`;
    localStorage.setItem(key, content);
    loadRoomClipboardContent();

    setTimeout(() => {
        isProcessingAction = false;
    }, 500); // Small delay to prevent rapid sharing
}

// Load shared clipboard content for room
function loadRoomClipboardContent() {
    if (!currentUser || !currentUser.roomId) return;

    const key = `room_${currentUser.roomId}_clipboard`;
    const content = localStorage.getItem(key) || 'No shared content yet...';

    const clipboardDiv = document.getElementById('room-clipboard-content');
    if (clipboardDiv) {
        clipboardDiv.innerHTML = `<pre>${content}</pre>`;
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', initApp);
