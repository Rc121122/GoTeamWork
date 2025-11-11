import { cleanup } from "./state";
import { hostListRooms, hostListUsers } from "./api/wailsBridge";
import { clearChildren } from "./ui/dom";
import { createRoomListItem, createUserListItem } from "./ui/templates";

let hostUpdateTimer: number | null = null;

export function renderHostLobby(): void {
  cleanup();

  const appRoot = document.querySelector<HTMLElement>("#app");
  if (!appRoot) {
    console.error("App root not found");
    return;
  }

  appRoot.innerHTML = `
    <div class="host-lobby">
      <div class="lobby-header">
        <h1>Server Management Lobby</h1>
        <p>Monitor users and rooms</p>
      </div>

      <div class="lobby-content">
        <div class="users-section">
          <h2>All Users</h2>
          <div id="all-users-list" class="users-list"></div>
        </div>

        <div class="rooms-section">
          <h2>All Rooms</h2>
          <div id="all-rooms-list" class="rooms-list"></div>
        </div>
      </div>
    </div>
  `;

  if (hostUpdateTimer !== null) {
    window.clearInterval(hostUpdateTimer);
  }

  void updateHostLobby();
  hostUpdateTimer = window.setInterval(() => {
    void updateHostLobby();
  }, 5000);
}

async function updateHostLobby(): Promise<void> {
  await Promise.all([updateUserList(), updateRoomList()]);
}

async function updateUserList(): Promise<void> {
  try {
    const users = await hostListUsers();
    const container = document.getElementById("all-users-list");
    if (!container) {
      return;
    }

    clearChildren(container);

    if (users.length === 0) {
      container.innerHTML = '<p class="empty">No users connected</p>';
      return;
    }

    users.forEach((user) => {
      container.appendChild(createUserListItem(user));
    });
  } catch (error) {
    console.error("Failed to update users list", error);
  }
}

async function updateRoomList(): Promise<void> {
  try {
    const rooms = await hostListRooms();
    const container = document.getElementById("all-rooms-list");
    if (!container) {
      return;
    }

    clearChildren(container);

    if (rooms.length === 0) {
      container.innerHTML = '<p class="empty">No active rooms</p>';
      return;
    }

    rooms.forEach((room) => {
      container.appendChild(createRoomListItem(room));
    });
  } catch (error) {
    console.error("Failed to update rooms list", error);
  }
}
