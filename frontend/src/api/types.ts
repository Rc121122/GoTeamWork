export interface User {
  id: string;
  name: string;
  roomId?: string | null;
  isOnline: boolean;
}

export interface Room {
  id: string;
  name: string;
  userIds: string[];
}

export interface ChatMessage {
  id: string;
  roomId: string;
  userId: string;
  userName: string;
  message: string;
  timestamp: number;
}

export interface ApiMessageResponse {
  message: string;
  roomId?: string;
}

export interface InviteEventPayload {
  roomId: string;
  roomName: string;
  inviter: string;
}

export interface SSEEnvelope<T> {
  type: string;
  data: T;
  timestamp: number;
}

export interface CreateUserRequest {
  name: string;
}

export interface InviteUserRequest {
  userId: string;
  inviterId: string;
}

export interface ChatMessageRequest {
  roomId: string;
  userId: string;
  message: string;
}

export interface LeaveRoomRequest {
  userId: string;
}

export interface JoinRoomRequest {
  userId: string;
  roomId: string;
}

export interface CreateRoomRequest {
  name: string;
}

export interface CopiedItem {
  type: "text" | "image";
  text?: string;
  image?: string; // base64 encoded
}

export type AppMode = "host" | "client";
