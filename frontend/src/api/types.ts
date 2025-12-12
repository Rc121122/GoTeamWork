export interface User {
  id: string;
  name: string;
  roomId?: string | null;
  isOnline: boolean;
}

export interface Room {
  id: string;
  name: string;
  ownerId?: string;
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
  inviteId?: string;
  expiresAt?: number;
}

export interface InviteEventPayload {
  inviteId: string;
  inviterId: string;
  inviter: string;
  message: string;
  expiresAt: number;
}

export interface SSEEnvelope<T> {
  type: string;
  data: T;
  timestamp: number;
}

export interface CreateUserRequest {
  name: string;
}

export interface CreateUserResponse {
  user: User;
  token: string;
}

export interface InviteUserRequest {
  userId: string;
  inviterId: string;
  message: string;
}

export interface AcceptInviteRequest {
  inviteId: string;
  inviteeId: string;
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
  type: "text" | "image" | "file";
  text?: string;
  image?: string; // base64 encoded
  files?: string[]; // file paths
}

export interface Operation {
  id: string;
  parentId: string;
  opType: string;
  itemId: string;
  item: {
    id: string;
    type: string;
    data: ChatMessage | CopiedItem;
  };
  timestamp: number;
  userId?: string;
  userName?: string;
}

export type AppMode = "host" | "client";
