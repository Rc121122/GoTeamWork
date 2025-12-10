import { globalState } from "./state";
import { connectSSE } from "./sse";
import { showError, clearChildren } from "./ui/dom";
import { createChatMessageElement, createUserListItem } from "./ui/templates";
import {
  HttpError,
  httpCreateUser,
  httpFetchChatHistory,
  httpFetchUsers,
  httpAcceptInvite,
  httpInviteUser,
  httpLeaveRoom,
  httpSendChatMessage,
} from "./api/httpClient";
import type { ChatMessage, CopiedItem, InviteEventPayload, User } from "./api/types";

let activeInviteTarget: User | null = null;

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
        onUserOffline: () => {
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

  const modal = document.getElementById("invite-modal");
  const acceptBtn = document.getElementById("accept-invite");
  const declineBtn = document.getElementById("decline-invite");
  const messageDiv = document.getElementById("invite-message");

  if (!modal || !acceptBtn || !declineBtn || !messageDiv) {
    return;
  }

  messageDiv.innerHTML = `<strong>${payload.inviter}</strong> says:<br/>${payload.message}`;
  (modal as HTMLElement).style.display = "flex";

  acceptBtn.onclick = () => {
    void acceptInvite(payload);
  };

  declineBtn.onclick = () => {
    (modal as HTMLElement).style.display = "none";
  };
}

async function acceptInvite(payload: InviteEventPayload): Promise<void> {
  const currentUser = globalState.currentUser;
  const modal = document.getElementById("invite-modal");
  const acceptBtn = document.getElementById("accept-invite");

  if (!currentUser || !modal || !acceptBtn) {
    return;
  }

  if (globalState.isProcessingAction) {
    return;
  }

  globalState.isProcessingAction = true;
  acceptBtn.setAttribute("disabled", "true");

  try {
    await httpAcceptInvite({ inviteId: payload.inviteId, inviteeId: currentUser.id });
    (modal as HTMLElement).style.display = "none";
  } catch (error) {
    console.error("Failed to accept invite", error);
    window.alert("Unable to accept invite. It may have expired.");
  } finally {
    globalState.isProcessingAction = false;
    acceptBtn.removeAttribute("disabled");
  }
}

function handleUserJoined(payload: { roomId: string; roomName: string; userId: string; userName: string }): void {
  console.log("User joined event:", payload);

  const currentUser = globalState.currentUser;
  if (!currentUser) {
    return;
  }

  if (payload.userId === currentUser.id) {
    globalState.currentUser = { ...currentUser, roomId: payload.roomId };
    clearPendingInvite();
    renderRoomView();
    return;
  }

  if (currentUser.roomId === payload.roomId) {
    console.log(`${payload.userName} joined your room.`);
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

    <div id="invite-compose-modal" class="modal" style="display: none;">
      <div class="modal-content">
        <h2>Customize Invite</h2>
        <p id="invite-target-name">Send a quick note.</p>
        <textarea id="invite-message-input" class="invite-message-input" maxlength="280"></textarea>
        <div class="modal-buttons">
          <button id="send-invite-message" class="accept-btn">Send Invite</button>
          <button id="cancel-invite-message" class="decline-btn">Cancel</button>
        </div>
      </div>
    </div>

    <div id="invite-pending-modal" class="modal" style="display: none;">
      <div class="modal-content pending-modal">
        <h2>Waiting for response</h2>
        <p id="pending-invite-text"></p>
        <div class="invite-countdown">
          <span id="invite-countdown">30</span>s remaining
        </div>
      </div>
    </div>
  `;

  setupInviteModals();
  void updateUserList();
}

function setupInviteModals(): void {
  const sendButton = document.getElementById("send-invite-message");
  const cancelButton = document.getElementById("cancel-invite-message");

  sendButton?.addEventListener("click", () => {
    if (!activeInviteTarget) {
      return;
    }

    const textarea = document.getElementById("invite-message-input") as HTMLTextAreaElement | null;
    const message = textarea?.value.trim() || buildDefaultInviteMessage();
    void sendInviteRequest(activeInviteTarget, message);
  });

  cancelButton?.addEventListener("click", () => {
    closeInviteComposeModal();
  });
}

function buildDefaultInviteMessage(): string {
  const name = globalState.currentUser?.name ?? "me";
  return `Hi, it's me, ${name}.`;
}

function openInviteComposeModal(target: User): void {
  if (globalState.pendingInvite) {
    window.alert("You already have a pending invite.");
    return;
  }

  const modal = document.getElementById("invite-compose-modal");
  const targetName = document.getElementById("invite-target-name");
  const textarea = document.getElementById("invite-message-input") as HTMLTextAreaElement | null;

  if (!modal || !targetName || !textarea) {
    return;
  }

  activeInviteTarget = target;
  targetName.textContent = `Invite ${target.name}`;
  textarea.value = buildDefaultInviteMessage();
  (modal as HTMLElement).style.display = "flex";
  window.setTimeout(() => textarea.focus(), 0);
}

function closeInviteComposeModal(): void {
  const modal = document.getElementById("invite-compose-modal");
  if (modal) {
    (modal as HTMLElement).style.display = "none";
  }
  activeInviteTarget = null;
}

async function sendInviteRequest(target: User, message: string): Promise<void> {
  const currentUser = globalState.currentUser;
  if (!currentUser) {
    window.alert("You must be logged in to invite users");
    return;
  }

  closeInviteComposeModal();

  try {
    const response = await httpInviteUser({ userId: target.id, inviterId: currentUser.id, message });
    if (!response.inviteId) {
      window.alert(response.message);
      return;
    }
    startPendingInviteState(response.inviteId, target, response.expiresAt);
    window.alert(response.message);
  } catch (error) {
    console.error("Error inviting user", error);
    window.alert("Failed to send invitation");
  }
}

function startPendingInviteState(inviteId: string, invitee: User, expiresAt?: number): void {
  clearPendingInvite();

  const modal = document.getElementById("invite-pending-modal");
  const countdown = document.getElementById("invite-countdown");
  const text = document.getElementById("pending-invite-text");

  if (!modal || !countdown || !text) {
    return;
  }

  const now = Math.floor(Date.now() / 1000);
  const fallbackExpiry = now + 30;
  const targetExpiry = expiresAt && expiresAt > 0 ? expiresAt : fallbackExpiry;
  let remaining = Math.max(0, targetExpiry - now);

  text.textContent = `Waiting for ${invitee.name} to respond...`;
  countdown.textContent = remaining.toString().padStart(2, "0");
  (modal as HTMLElement).style.display = "flex";

  const timerId = window.setInterval(() => {
    remaining -= 1;
    countdown.textContent = Math.max(0, remaining).toString().padStart(2, "0");

    if (remaining <= 0) {
      clearPendingInvite(true);
    }
  }, 1000);

  globalState.pendingInvite = {
    inviteId,
    inviteeId: invitee.id,
    expiresAt: targetExpiry,
    countdownTimerId: timerId,
  };

  void updateUserList();
}

function clearPendingInvite(expired = false): void {
  if (globalState.pendingInvite?.countdownTimerId) {
    window.clearInterval(globalState.pendingInvite.countdownTimerId);
  }

  globalState.pendingInvite = null;

  const modal = document.getElementById("invite-pending-modal");
  if (modal) {
    (modal as HTMLElement).style.display = "none";
  }

  if (expired) {
    window.alert("Invite expired without response.");
  }

  void updateUserList();
}

function promptInviteMessage(user: User): void {
  openInviteComposeModal(user);
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
  const allowInvites = !globalState.pendingInvite;

  if (otherUsers.length === 0) {
    userListDiv.innerHTML = '<p class="empty">No other users online</p>';
    return;
  }

  otherUsers.forEach((user) => {
    const item = createUserListItem(user, {
      showInviteButton: allowInvites && !user.roomId,
      onInvite: promptInviteMessage,
    });
    userListDiv.appendChild(item);
  });
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
