// common.js - Shared utilities and SSE connection logic

// Global state - using an object to allow mutation
export const globalState = {
    currentUser: null,
    currentRoom: null,
    sseConnection: null,
    isProcessingAction: false
};

// Cleanup function to clear SSE connection
export function cleanup() {
    if (globalState.sseConnection) {
        globalState.sseConnection.close();
        globalState.sseConnection = null;
    }
    globalState.isProcessingAction = false;
}

// Connect to SSE after user creation
export function connectSSE(userId, onUserCreated, onUserInvited, onChatMessage) {
    const sseUrl = `http://localhost:8080/api/sse?userId=${userId}`;
    globalState.sseConnection = new EventSource(sseUrl);

    globalState.sseConnection.onopen = function(event) {
        console.log('SSE connected');
    };

    globalState.sseConnection.onmessage = function(event) {
        console.log('SSE message received:', event.data);
    };

    globalState.sseConnection.addEventListener('user_created', function(event) {
        const data = JSON.parse(event.data);
        console.log('New user created:', data.data);
        if (onUserCreated) onUserCreated(data.data);
    });

    globalState.sseConnection.addEventListener('user_invited', function(event) {
        const data = JSON.parse(event.data);
        if (onUserInvited) onUserInvited(data.data);
    });

    globalState.sseConnection.addEventListener('chat_message', function(event) {
        const data = JSON.parse(event.data);
        if (onChatMessage) onChatMessage(data.data);
    });

    globalState.sseConnection.addEventListener('connected', function(event) {
        console.log('SSE connection confirmed');
    });

    globalState.sseConnection.addEventListener('heartbeat', function(event) {
        console.log('SSE heartbeat received');
    });

    globalState.sseConnection.onerror = function(error) {
        console.error('SSE error:', error);
        // Auto-reconnect after 5 seconds
        setTimeout(() => connectSSE(userId, onUserCreated, onUserInvited, onChatMessage), 5000);
    };
}

// Utility function to show error messages
export function showError(elementId, message, duration = 5000) {
    const errorDiv = document.getElementById(elementId);
    if (errorDiv) {
        errorDiv.textContent = message;
        errorDiv.style.display = 'block';
        setTimeout(() => {
            errorDiv.style.display = 'none';
        }, duration);
    }
}

// Utility function to create user item HTML
export function createUserItemHTML(user, showInviteButton = false, inviteCallback = null) {
    const roomStatus = user.roomId ? `In room: ${user.roomId}` : 'Not in room';

    let inviteButton = '';
    if (showInviteButton && !user.roomId) {
        inviteButton = `<button class="invite-btn" onclick="${inviteCallback ? inviteCallback(user.id, user.name) : `inviteUser('${user.id}', '${user.name}')`}">Invite</button>`;
    }

    return `
        <div class="user-info">
            <span class="user-name">${user.name}</span>
            <span class="user-status ${user.isOnline ? 'online' : 'offline'}">
                ● ${user.isOnline ? 'Online' : 'Offline'}
            </span>
            <span class="user-room">${roomStatus}</span>
            ${inviteButton}
        </div>
    `;
}

// Utility function to create room item HTML
export function createRoomItemHTML(room) {
    return `
        <div class="room-info">
            <span class="room-name">${room.name}</span>
            <span class="room-users">Users: ${room.userIds.length}</span>
            <div class="room-user-list">
                ${room.userIds.join(', ')}
            </div>
        </div>
    `;
}

// Utility function to create chat message HTML
export function createChatMessageHTML(message) {
    const timestamp = message.timestamp ? new Date(message.timestamp * 1000).toLocaleTimeString() : new Date().toLocaleTimeString();

    return `
        <div class="message-header">
            <span class="message-user">${message.userName}</span>
            <span class="message-time">${timestamp}</span>
        </div>
        <div class="message-content">${message.message}</div>
    `;
}

// API call utilities
export async function apiCall(endpoint, method = 'GET', body = null) {
    const config = {
        method,
        headers: {
            'Content-Type': 'application/json',
        },
    };

    if (body) {
        config.body = JSON.stringify(body);
    }

    const response = await fetch(`http://localhost:8080${endpoint}`, config);
    return response;
}