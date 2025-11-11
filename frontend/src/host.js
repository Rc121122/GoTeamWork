// host.js - Host mode specific functionality

import { cleanup, createUserItemHTML, createRoomItemHTML, apiCall } from './common.js';

// Render host lobby (shows all users and rooms for monitoring)
export function renderHostLobby() {
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

    // Start updating user and room lists
    updateHostLobby();
    setInterval(updateHostLobby, 5000);
}

// Update host lobby with current users and rooms
async function updateHostLobby() {
    try {
        // Update users list
        const usersResponse = await apiCall('/api/users');
        if (usersResponse.ok) {
            const users = await usersResponse.json();
            renderHostUsersList(users);
        }

        // Update rooms list
        const roomsResponse = await apiCall('/api/rooms');
        if (roomsResponse.ok) {
            const rooms = await roomsResponse.json();
            renderHostRoomsList(rooms);
        }
    } catch (error) {
        console.error('Failed to update host lobby:', error);
    }
}

// Render users list for host (no invite buttons)
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

        // Host mode: no invite buttons, just monitoring
        userDiv.innerHTML = createUserItemHTML(user, false);

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

        roomDiv.innerHTML = createRoomItemHTML(room);

        roomsListDiv.appendChild(roomDiv);
    });
}