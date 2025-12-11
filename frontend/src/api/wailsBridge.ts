import {
  CreateRoom,
  CreateUser,
  GetAllRooms,
  GetChatHistory,
  GetMode,
  GetOperations,
  Invite,
  JoinRoom,
  LeaveRoom,
  ListAllUsers,
  SendChatMessage,
  SetServerURL,
  SetUser,
} from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";
import type { AppMode, ChatMessage, Room, User, Operation } from "./types";

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
    ownerId: (room as any).ownerId, // Cast to any because bindings might not be updated yet
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

export async function hostSetUser(id: string, name: string): Promise<User> {
  const user = await SetUser(id, name);
  return mapUser(user);
}

export async function hostCreateRoom(name: string): Promise<Room> {
  const room = await CreateRoom(name);
  return mapRoom(room);
}

export async function hostInviteUser(userId: string): Promise<string> {
  return Invite(userId);
}

export async function hostJoinRoom(roomId: string, userId: string): Promise<Room> {
  const room = await JoinRoom(userId, roomId); // Note: Go method is JoinRoom(userID, roomID)
  return mapRoom(room);
}

export async function hostRequestJoin(userId: string, roomId: string): Promise<string> {
  // @ts-ignore
  return window.go.main.App.RequestJoinRoom(userId, roomId);
}

export async function hostApproveJoin(ownerId: string, requesterId: string, roomId: string): Promise<void> {
  // @ts-ignore
  return window.go.main.App.ApproveJoinRequest(ownerId, requesterId, roomId);
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

export async function hostFetchOperations(roomId: string, sinceId: string = ""): Promise<Operation[]> {
  const ops = await GetOperations(roomId, sinceId);
  // @ts-ignore
  return ops.map(op => ({
      id: op.id,
      parentId: op.parentId,
      opType: op.opType,
      itemId: op.itemId,
      item: op.item ? {
          id: op.item.id,
          type: op.item.type,
          data: op.item.data
      } : { id: "unknown", type: "unknown", data: {} as any },
      timestamp: op.timestamp
  }));
}


export function shareSystemClipboard(): Promise<boolean> {
  // @ts-ignore
  return window.go?.main?.App?.ShareSystemClipboard?.() ?? Promise.resolve(false);
}

export async function hostSetServerURL(url: string): Promise<void> {
  await SetServerURL(url);
}
