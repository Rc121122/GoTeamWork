// client.js - Client mode specific functionality

import { globalState, cleanup, connectSSE, showError, createUserItemHTML, createChatMessageHTML, apiCall } from './common.js';

// Render client UI (username input -> waiting lobby)
export function renderClientUI() {
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

    input.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            joinWithUsername();
        }
    });

    button.addEventListener('click', joinWithUsername);

    async function joinWithUsername() {
        const username = input.value.trim();
        if (!username) {
            showError('username-error', 'Please enter a username');
            return;
        }

        try {
            // Try to create user via HTTP API
            const response = await apiCall('/api/users', 'POST', { name: username });

            if (response.ok) {
                const user = await response.json();
                globalState.currentUser = user;
                connectSSE(user.id, onUserCreated, onUserInvited, onChatMessage);
                renderWaitingLobby();
            } else if (response.status === 409) {
                showError('username-error', 'Username already exists. Please choose another one.');
            } else {
                showError('username-error', 'Failed to join. Please try again.');
            }
        } catch (error) {
            console.error('Error joining:', error);
            showError('username-error', 'Cannot connect to server. Please check if host is running.');
        }
    }
}

// SSE event handlers
function onUserCreated(user) {
    // Refresh user list
    updateUserList();
}

function onUserInvited(data) {
    showInviteModal(data.roomId, data.roomName, data.inviter);
}

function onChatMessage(data) {
    // Add new message to chat without full refresh
    renderRoomChatMessage(data);
}

// Render waiting lobby
function renderWaitingLobby() {
    // Cleanup previous intervals
    cleanup();

    document.querySelector('#app').innerHTML = `
        <div class="waiting-lobby">
            <div class="lobby-header">
                <h1>Waiting Lobby</h1>
                <p>Welcome, ${globalState.currentUser.name}! Waiting for others to join...</p>
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

    // Load initial user list (SSE will handle updates)
    updateUserList();
}

// Update user list from server
async function updateUserList() {
    try {
        const response = await apiCall('/api/users');
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

// Render user list in waiting lobby (with invite buttons)
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
    const otherUsers = users.filter(user => user && user.id !== globalState.currentUser?.id);

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

        // Client mode: show invite buttons for users not in rooms
        userDiv.innerHTML = createUserItemHTML(user, !user.roomId, inviteUser);

        userListDiv.appendChild(userDiv);
    });
}

// Invite a user
async function inviteUser(userId, userName) {
    try {
        const response = await apiCall('/api/invite', 'POST', { userId: userId });

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

// Show invite modal
function showInviteModal(roomId, roomName, inviter) {
    const modal = document.getElementById('invite-modal');
    const acceptBtn = document.getElementById('accept-invite');
    const declineBtn = document.getElementById('decline-invite');
    const messageDiv = document.getElementById('invite-message');

    // Update message with room details
    if (roomName && inviter) {
        messageDiv.textContent = `${inviter} has invited you to join room "${roomName}"!`;
    } else {
        messageDiv.textContent = 'You have been invited to join a room!';
    }

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
    if (!globalState.currentUser || !globalState.currentUser.roomId) return;

    try {
        const response = await apiCall('/api/leave', 'POST', { userId: globalState.currentUser.id });

        if (response.ok) {
            const result = await response.json();
            console.log('Left room:', result.message);
            globalState.currentUser.roomId = null;
            renderWaitingLobby();
        } else {
            console.error('Failed to leave room, status:', response.status);
        }
    } catch (error) {
        console.error('Error leaving room:', error);
        // Fallback: just update local state
        globalState.currentUser.roomId = null;
        renderWaitingLobby();
    }
}

// Render room view (chat + clipboard)
function renderRoomView() {
    if (!globalState.currentUser || !globalState.currentUser.roomId) {
        renderWaitingLobby();
        return;
    }

    // Cleanup previous intervals
    cleanup();

    document.querySelector('#app').innerHTML = `
        <div class="room-view">
            <div class="room-header">
                <h1>Room: ${globalState.currentUser.roomId}</h1>
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

    // Load clipboard content
    loadRoomClipboardContent();

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

    // SSE will handle real-time chat updates
}

// Send message to room
async function sendRoomMessage(message) {
    if (!globalState.currentUser || !globalState.currentUser.roomId || globalState.isProcessingAction) return;

    globalState.isProcessingAction = true;

    try {
        const response = await apiCall('/api/chat', 'POST', {
            roomId: globalState.currentUser.roomId,
            userId: globalState.currentUser.id,
            message: message,
        });

        if (response.ok) {
            loadRoomChatHistory(); // Refresh messages
        } else {
            console.error('Failed to send message, status:', response.status);
        }
    } catch (error) {
        console.error('Error sending message:', error);
    } finally {
        globalState.isProcessingAction = false;
    }
}

// Load chat history for current room
async function loadRoomChatHistory() {
    if (!globalState.currentUser || !globalState.currentUser.roomId) return;

    try {
        const response = await apiCall(`/api/chat/${globalState.currentUser.roomId}`);
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

        messageDiv.innerHTML = createChatMessageHTML(message);

        chatDiv.appendChild(messageDiv);
    });

    // Scroll to bottom
    chatDiv.scrollTop = chatDiv.scrollHeight;
}

// Render individual chat message (for SSE updates)
function renderRoomChatMessage(message) {
    const chatDiv = document.getElementById('room-chat-messages');
    if (!chatDiv) return;

    const messageDiv = document.createElement('div');
    messageDiv.className = 'chat-message';

    messageDiv.innerHTML = createChatMessageHTML(message);

    chatDiv.appendChild(messageDiv);
    chatDiv.scrollTop = chatDiv.scrollHeight; // Scroll to bottom
}

// Share clipboard content in room
function shareRoomClipboard(content) {
    if (globalState.isProcessingAction) return;

    globalState.isProcessingAction = true;

    // For now, store in localStorage with room prefix
    // In a real implementation, this would be sent to the server
    const key = `room_${globalState.currentUser.roomId}_clipboard`;
    localStorage.setItem(key, content);
    loadRoomClipboardContent();

    setTimeout(() => {
        globalState.isProcessingAction = false;
    }, 500); // Small delay to prevent rapid sharing
}

// Load clipboard content for current room
function loadRoomClipboardContent() {
    if (!globalState.currentUser || !globalState.currentUser.roomId) return;

    const key = `room_${globalState.currentUser.roomId}_clipboard`;
    const content = localStorage.getItem(key);

    const clipboardDiv = document.getElementById('room-clipboard-content');
    if (!clipboardDiv) return;

    if (content) {
        clipboardDiv.innerHTML = `<pre>${content}</pre>`;
    } else {
        clipboardDiv.innerHTML = '<p class="empty">No shared content yet</p>';
    }
}