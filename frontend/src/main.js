import './style.css';
import './app.css';

import { GetMode } from '../wailsjs/go/main/App';

// Global state
let currentMode = 'client';
let currentUser = null;
let userListInterval = null;

// Initialize the app
async function initApp() {
    try {
        currentMode = await GetMode();
        console.log('App mode:', currentMode);

        if (currentMode === 'host') {
            renderHostUI();
        } else {
            renderClientUI();
        }
    } catch (error) {
        console.error('Failed to get mode:', error);
        // Default to client mode
        renderClientUI();
    }
}

// Render host UI (existing chat/clipboard interface)
function renderHostUI() {
    document.querySelector('#app').innerHTML = `
        <div class="container">
            <div class="chat-room">
                <h2>Chat Room</h2>
                <div id="chat-messages"></div>
                <div id="chat-input">
                    <input type="text" id="message-input" placeholder="Type a message..." />
                    <button id="send-button">Send</button>
                </div>
            </div>
            <div class="clipboard-share">
                <h2>Clipboard Share</h2>
                <div id="clipboard-content"></div>
                <div id="clipboard-input">
                    <textarea id="clipboard-text" placeholder="Paste content to share..."></textarea>
                    <button id="share-button">Share</button>
                </div>
            </div>
        </div>
    `;

    // Load existing functionality
    loadChatMessages();
    loadClipboardContent();

    // Add event listeners
    document.getElementById('send-button').addEventListener('click', () => {
        const input = document.getElementById('message-input');
        const message = input.value.trim();
        if (message) {
            const messages = JSON.parse(localStorage.getItem('chatMessages') || '[]');
            messages.push(message);
            localStorage.setItem('chatMessages', JSON.stringify(messages));
            input.value = '';
            loadChatMessages();
        }
    });

    document.getElementById('share-button').addEventListener('click', () => {
        const textarea = document.getElementById('clipboard-text');
        const content = textarea.value.trim();
        if (content) {
            localStorage.setItem('clipboardContent', content);
            textarea.value = '';
            loadClipboardContent();
        }
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
    `;

    // Start polling for user updates
    updateUserList();
    userListInterval = setInterval(updateUserList, 5000);
}

// Update user list from server
async function updateUserList() {
    try {
        const response = await fetch('http://localhost:8080/api/users');
        if (response.ok) {
            const users = await response.json();
            renderUserList(users);
        }
    } catch (error) {
        console.error('Failed to fetch users:', error);
    }
}

// Render the user list with invite buttons
function renderUserList(users) {
    const userListDiv = document.getElementById('user-list');
    userListDiv.innerHTML = '';

    users.forEach(user => {
        if (user.id === currentUser.id) return; // Don't show current user

        const userDiv = document.createElement('div');
        userDiv.className = 'user-item';

        userDiv.innerHTML = `
            <div class="user-info">
                <span class="user-name">${user.name}</span>
                <span class="user-status ${user.isOnline ? 'online' : 'offline'}">
                    ${user.isOnline ? '● Online' : '● Offline'}
                </span>
            </div>
            <button class="invite-button" data-user-id="${user.id}">
                Invite
            </button>
        `;

        // Add invite button event listener
        const inviteButton = userDiv.querySelector('.invite-button');
        inviteButton.addEventListener('click', () => inviteUser(user.id, user.name));

        userListDiv.appendChild(userDiv);
    });
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

// Helper functions for host mode
function loadChatMessages() {
    const messages = JSON.parse(localStorage.getItem('chatMessages') || '[]');
    const chatMessagesDiv = document.getElementById('chat-messages');
    chatMessagesDiv.innerHTML = messages.map(msg => `<div>${msg}</div>`).join('');
}

function loadClipboardContent() {
    const content = localStorage.getItem('clipboardContent') || '';
    document.getElementById('clipboard-content').innerHTML = `<pre>${content}</pre>`;
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', initApp);
