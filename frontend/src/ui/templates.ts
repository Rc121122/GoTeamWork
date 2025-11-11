import type { ChatMessage, Room, User } from "../api/types";

function formatTimestamp(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleTimeString();
}

export function createUserListItem(
  user: User,
  options: { showInviteButton?: boolean; onInvite?: (user: User) => void } = {},
): HTMLDivElement {
  const { showInviteButton = false, onInvite } = options;

  const container = document.createElement("div");
  container.className = "user-item";

  const info = document.createElement("div");
  info.className = "user-info";

  const nameSpan = document.createElement("span");
  nameSpan.className = "user-name";
  nameSpan.textContent = user.name;

  const statusSpan = document.createElement("span");
  statusSpan.className = `user-status ${user.isOnline ? "online" : "offline"}`;
  statusSpan.textContent = `● ${user.isOnline ? "Online" : "Offline"}`;

  const roomSpan = document.createElement("span");
  roomSpan.className = "user-room";
  roomSpan.textContent = user.roomId ? `In room: ${user.roomId}` : "Not in room";

  info.append(nameSpan, statusSpan, roomSpan);

  if (showInviteButton && onInvite && !user.roomId) {
    const inviteButton = document.createElement("button");
    inviteButton.className = "invite-btn";
    inviteButton.textContent = "Invite";
    inviteButton.addEventListener("click", () => onInvite(user));
    info.append(inviteButton);
  }

  container.append(info);
  return container;
}

export function createRoomListItem(room: Room): HTMLDivElement {
  const container = document.createElement("div");
  container.className = "room-item";

  const info = document.createElement("div");
  info.className = "room-info";

  const nameSpan = document.createElement("span");
  nameSpan.className = "room-name";
  nameSpan.textContent = room.name;

  const userCountSpan = document.createElement("span");
  userCountSpan.className = "room-users";
  userCountSpan.textContent = `Users: ${room.userIds.length}`;

  const usersList = document.createElement("div");
  usersList.className = "room-user-list";
  usersList.textContent = room.userIds.join(", ");

  info.append(nameSpan, userCountSpan, usersList);
  container.append(info);
  return container;
}

export function createChatMessageElement(message: ChatMessage): HTMLDivElement {
  const container = document.createElement("div");
  container.className = "chat-message";

  const header = document.createElement("div");
  header.className = "message-header";

  const userSpan = document.createElement("span");
  userSpan.className = "message-user";
  userSpan.textContent = message.userName;

  const timeSpan = document.createElement("span");
  timeSpan.className = "message-time";
  timeSpan.textContent = formatTimestamp(message.timestamp);

  const content = document.createElement("div");
  content.className = "message-content";
  content.textContent = message.message;

  header.append(userSpan, timeSpan);
  container.append(header, content);

  return container;
}
