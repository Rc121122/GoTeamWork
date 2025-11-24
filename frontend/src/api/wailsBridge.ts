import {
  CreateRoom,
  CreateUser,
  GetAllRooms,
  GetChatHistory,
  GetMode,
  Invite,
  LeaveRoom,
  ListAllUsers,
  SendChatMessage,
  ShareSystemClipboard,
} from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";
import type { AppMode, ChatMessage, Room, User } from "./types";

function mapUser(user: main.User): User {
  return {
    id: user.id,
    name: user.name,
    roomId: user.roomId ?? null,
    isOnline: user.isOnline,
  };
}

function mapRoom(room: main.Room): Room {
  return {
    id: room.id,
    name: room.name,
    userIds: [...room.userIds],
  };
}

function mapChatMessage(message: main.ChatMessage): ChatMessage {
  return {
    id: message.id,
    roomId: message.roomId,
    userId: message.userId,
    userName: message.userName,
    message: message.message,
    timestamp: message.timestamp,
  };
}

export async function getAppMode(): Promise<AppMode> {
  const mode = await GetMode();
  if (mode === "host" || mode === "client") {
    return mode;
  }
  return "client";
}

export async function hostListUsers(): Promise<User[]> {
  const users = await ListAllUsers();
  return users.map(mapUser);
}

export async function hostListRooms(): Promise<Room[]> {
  const rooms = await GetAllRooms();
  return rooms.map(mapRoom);
}

export async function hostCreateUser(name: string): Promise<User> {
  const user = await CreateUser(name);
  return mapUser(user);
}

export async function hostCreateRoom(name: string): Promise<Room> {
  const room = await CreateRoom(name);
  return mapRoom(room);
}

export async function hostInviteUser(userId: string): Promise<string> {
  return Invite(userId);
}

export async function hostLeaveRoom(userId: string): Promise<string> {
  return LeaveRoom(userId);
}

export async function hostSendChatMessage(roomId: string, userId: string, message: string): Promise<string> {
  return SendChatMessage(roomId, userId, message);
}

export async function hostFetchChatHistory(roomId: string): Promise<ChatMessage[]> {
  const history = await GetChatHistory(roomId);
  return history.map(mapChatMessage);
}

export function shareSystemClipboard(): Promise<boolean> {
  return ShareSystemClipboard();
}
