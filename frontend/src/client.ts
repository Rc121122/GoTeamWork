import { globalState } from "./state";
import { connectSSE } from "./sse";
import { showError, clearChildren } from "./ui/dom";
import { createChatMessageElement, createUserListItem } from "./ui/templates";
import {
  HttpError,
  httpCreateUser,
  httpFetchChatHistory,
  httpFetchUsers,
  httpInviteUser,
  httpJoinRoom,
  httpLeaveRoom,
  httpSendChatMessage,
} from "./api/httpClient";
import type { ChatMessage, CopiedItem, InviteEventPayload, User } from "./api/types";

export function renderClientUI(): void {
  renderUsernameInput();
}

function renderUsernameInput(): void {
  const appRoot = document.querySelector<HTMLElement>("#app");
  if (!appRoot) {
    console.error("App root not found");
    return;
  }

  appRoot.innerHTML = `
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

  const input = document.getElementById("username-input") as HTMLInputElement | null;
  const button = document.getElementById("join-button");

  if (!input || !button) {
    console.error("Username input elements not found");
    return;
  }

  input.addEventListener("keypress", (event: KeyboardEvent) => {
    if (event.key === "Enter") {
      void joinWithUsername();
    }
  });

  button.addEventListener("click", () => {
    void joinWithUsername();
  });

  const joinWithUsername = async (): Promise<void> => {
    const username = input.value.trim();
    if (!username) {
      showError("username-error", "Please enter a username");
      return;
    }

    try {
      const user = await httpCreateUser({ name: username });
      globalState.currentUser = { ...user, roomId: user.roomId ?? null };

      connectSSE(user.id, {
        onUserCreated: () => {
          void updateUserList();
        },
        onUserInvited: handleInvite,
        onUserJoined: handleUserJoined,
        onChatMessage: renderRoomChatMessage,
        onClipboardCopied: handleClipboardCopied,
        onDisconnected: () => {
          console.warn("SSE connection lost; attempting to reconnect");
        },
      });

      renderWaitingLobby();
    } catch (error) {
      if (error instanceof HttpError && error.status === 409) {
        showError("username-error", "Username already exists. Please choose another one.");
      } else {
        console.error("Error joining", error);
        showError("username-error", "Cannot connect to server. Please check if host is running.");
      }
    }
  };
}

function handleInvite(payload: InviteEventPayload): void {
  console.log("Received invite event:", payload);
  
  // If inviter is "Self", this user sent the invite - auto-join
  if (payload.inviter === "Self") {
    console.log("Auto-joining room as inviter:", payload.roomId);
    if (globalState.currentUser) {
      globalState.currentUser.roomId = payload.roomId;
    }
    renderRoomView();
    return;
  }

  // Otherwise, show invite modal for the invitee
  console.log("Showing invite modal for invitee");
  const modal = document.getElementById("invite-modal");
  const acceptBtn = document.getElementById("accept-invite");
  const declineBtn = document.getElementById("decline-invite");
  const messageDiv = document.getElementById("invite-message");

  if (!modal || !acceptBtn || !declineBtn || !messageDiv) {
    return;
  }

  messageDiv.textContent = `${payload.inviter} has invited you to join room "${payload.roomName}"!`;
  (modal as HTMLElement).style.display = "flex";

  acceptBtn.onclick = () => {
    (modal as HTMLElement).style.display = "none";
    if (globalState.currentUser) {
      globalState.currentUser.roomId = payload.roomId;
      // Notify the server that this user has joined the room
      void joinRoom(globalState.currentUser.id, payload.roomId);
    }
    renderRoomView();
  };

  declineBtn.onclick = () => {
    (modal as HTMLElement).style.display = "none";
    void leaveRoom();
  };
}

function handleUserJoined(payload: { roomId: string; roomName: string; userId: string; userName: string }): void {
  console.log("User joined event:", payload);
  
  // If this event is for our current room, refresh the view or show notification
  if (globalState.currentUser?.roomId === payload.roomId) {
    console.log(`${payload.userName} joined the room!`);
    // You could show a notification here or update a user list in the room
  }
}

function handleClipboardCopied(item: CopiedItem): void {
  const notification = document.getElementById("notification");
  if (!notification) {
    return;
  }

  let message = "";
  switch (item.type) {
    case "text":
      message = "Text copied to shareboard!";
      break;
    case "image":
      message = "Image copied to shareboard!";
      break;
    default:
      message = "Item copied to shareboard!";
  }

  notification.textContent = message;
  notification.style.display = "block";

  // Hide after 3 seconds
  setTimeout(() => {
    notification.style.display = "none";
  }, 3000);
}

function renderWaitingLobby(): void {
  const appRoot = document.querySelector<HTMLElement>("#app");
  if (!appRoot) {
    console.error("App root not found");
    return;
  }

  globalState.isProcessingAction = false;

  appRoot.innerHTML = `
    <div class="waiting-lobby">
      <div class="lobby-header">
        <h1>Waiting Lobby</h1>
        <p>Welcome, ${globalState.currentUser?.name ?? ""}! Waiting for others to join...</p>
      </div>
      <div class="user-list-container">
        <h2>Online Users</h2>
        <div id="user-list" class="user-list"></div>
      </div>
    </div>

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

  void updateUserList();
}

async function updateUserList(): Promise<void> {
  try {
    const users = await httpFetchUsers();
    renderUserList(users);
  } catch (error) {
    console.error("Failed to fetch users", error);
  }
}

function renderUserList(users: User[]): void {
  const userListDiv = document.getElementById("user-list");
  if (!userListDiv) {
    return;
  }

  clearChildren(userListDiv);

  const otherUsers = users.filter((user) => user.id !== globalState.currentUser?.id);

  if (otherUsers.length === 0) {
    userListDiv.innerHTML = '<p class="empty">No other users online</p>';
    return;
  }

  otherUsers.forEach((user) => {
    const item = createUserListItem(user, {
      showInviteButton: !user.roomId,
      onInvite: (target) => {
        void inviteUser(target.id, target.name);
      },
    });
    userListDiv.appendChild(item);
  });
}

async function inviteUser(userId: string, userName: string): Promise<void> {
  const currentUser = globalState.currentUser;
  if (!currentUser) {
    window.alert("You must be logged in to invite users");
    return;
  }

  console.log(`Inviting user ${userName} (${userId}) from ${currentUser.name} (${currentUser.id})`);

  try {
    const response = await httpInviteUser({ 
      userId,
      inviterId: currentUser.id 
    });
    
    console.log(`Invite response:`, response);
    
    // If roomId is returned, update current user and render room view
    if (response.roomId) {
      console.log(`Updating currentUser.roomId to ${response.roomId}`);
      globalState.currentUser = { ...currentUser, roomId: response.roomId };
      console.log(`Waiting for SSE event to render room view...`);
      // Note: We'll receive SSE event to actually render the room
      // The SSE event handler will call renderRoomView()
    }
    
    window.alert(response.message);
  } catch (error) {
    console.error("Error inviting user", error);
    window.alert("Failed to send invitation");
  }
}

async function joinRoom(userId: string, roomId: string): Promise<void> {
  try {
    const response = await httpJoinRoom({ userId, roomId });
    console.log("Join room response:", response);
  } catch (error) {
    console.error("Error joining room", error);
  }
}

async function leaveRoom(): Promise<void> {
  const currentUser = globalState.currentUser;
  if (!currentUser) {
    return;
  }

  if (!currentUser.roomId) {
    renderWaitingLobby();
    return;
  }

  try {
    const response = await httpLeaveRoom({ userId: currentUser.id });
    console.log("Left room", response.message);
  } catch (error) {
    console.error("Error leaving room", error);
  } finally {
    globalState.currentUser = { ...currentUser, roomId: null };
    renderWaitingLobby();
  }
}

function renderRoomView(): void {
  const currentUser = globalState.currentUser;
  if (!currentUser || !currentUser.roomId) {
    renderWaitingLobby();
    return;
  }

  const appRoot = document.querySelector<HTMLElement>("#app");
  if (!appRoot) {
    console.error("App root not found");
    return;
  }

  appRoot.innerHTML = `
    <div class="room-view">
      <div class="room-header">
        <h1>Room: ${currentUser.roomId}</h1>
        <button id="leave-room-btn" class="leave-btn">Leave Room</button>
      </div>

      <div class="room-content">
        <div class="chat-section">
          <h2>Chat</h2>
          <div id="room-chat-messages" class="chat-messages"></div>
          <div class="chat-input-area">
            <input type="text" id="room-message-input" placeholder="Type a message..." />
            <button id="room-send-button">Send</button>
          </div>
        </div>

        <div class="clipboard-section">
          <h2>Shared Clipboard</h2>
          <div id="room-clipboard-content" class="clipboard-content"></div>
          <div class="clipboard-input-area">
            <textarea id="room-clipboard-text" placeholder="Paste content to share..."></textarea>
            <button id="room-share-button">Share</button>
          </div>
        </div>
      </div>
    </div>
  `;

  const leaveButton = document.getElementById("leave-room-btn");
  const sendButton = document.getElementById("room-send-button");
  const shareButton = document.getElementById("room-share-button");

  leaveButton?.addEventListener("click", () => {
    void leaveRoom();
  });

  sendButton?.addEventListener("click", () => {
    const input = document.getElementById("room-message-input") as HTMLInputElement | null;
    if (!input) {
      return;
    }
    const content = input.value.trim();
    if (content) {
      void sendRoomMessage(content);
      input.value = "";
    }
  });

  shareButton?.addEventListener("click", () => {
    const textarea = document.getElementById("room-clipboard-text") as HTMLTextAreaElement | null;
    if (!textarea) {
      return;
    }
    const content = textarea.value.trim();
    if (content) {
      shareRoomClipboard(content);
      textarea.value = "";
    }
  });

  void loadRoomChatHistory();
  loadRoomClipboardContent();
}

async function sendRoomMessage(message: string): Promise<void> {
  const currentUser = globalState.currentUser;
  if (!currentUser || !currentUser.roomId || globalState.isProcessingAction) {
    return;
  }

  globalState.isProcessingAction = true;

  try {
    await httpSendChatMessage({
      roomId: currentUser.roomId,
      userId: currentUser.id,
      message,
    });
    await loadRoomChatHistory();
  } catch (error) {
    console.error("Error sending chat message", error);
  } finally {
    globalState.isProcessingAction = false;
  }
}

async function loadRoomChatHistory(): Promise<void> {
  const currentUser = globalState.currentUser;
  if (!currentUser || !currentUser.roomId) {
    return;
  }

  try {
    const messages = await httpFetchChatHistory(currentUser.roomId);
    renderRoomChatMessages(messages);
  } catch (error) {
    console.error("Failed to load chat history", error);
  }
}

function renderRoomChatMessages(messages: ChatMessage[]): void {
  const container = document.getElementById("room-chat-messages");
  if (!container) {
    return;
  }

  clearChildren(container);
  messages.forEach((message) => {
    container.appendChild(createChatMessageElement(message));
  });
  container.scrollTop = container.scrollHeight;
}

function renderRoomChatMessage(message: ChatMessage): void {
  const container = document.getElementById("room-chat-messages");
  if (!container || message.roomId !== globalState.currentUser?.roomId) {
    return;
  }

  container.appendChild(createChatMessageElement(message));
  container.scrollTop = container.scrollHeight;
}

function shareRoomClipboard(content: string): void {
  if (!globalState.currentUser || !globalState.currentUser.roomId) {
    return;
  }

  if (globalState.isProcessingAction) {
    return;
  }

  globalState.isProcessingAction = true;
  const key = `room_${globalState.currentUser.roomId}_clipboard`;
  localStorage.setItem(key, content);
  loadRoomClipboardContent();

  window.setTimeout(() => {
    globalState.isProcessingAction = false;
  }, 500);
}

function loadRoomClipboardContent(): void {
  if (!globalState.currentUser || !globalState.currentUser.roomId) {
    return;
  }

  const key = `room_${globalState.currentUser.roomId}_clipboard`;
  const content = localStorage.getItem(key);
  const container = document.getElementById("room-clipboard-content");

  if (!container) {
    return;
  }

  if (content) {
    container.innerHTML = `<pre>${content}</pre>`;
  } else {
    container.innerHTML = '<p class="empty">No shared content yet</p>';
  }
}
