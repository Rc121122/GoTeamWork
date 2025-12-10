import type {
  ApiMessageResponse,
  AcceptInviteRequest,
  ChatMessage,
  ChatMessageRequest,
  CreateUserRequest,
  InviteUserRequest,
  JoinRoomRequest,
  LeaveRoomRequest,
  User,
  Room,
} from "./types";

const API_BASE_URL = "http://localhost:8080";

export class HttpError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "HttpError";
    Object.setPrototypeOf(this, HttpError.prototype);
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      "Content-Type": "application/json",
    },
    ...init,
  });

  if (!response.ok) {
    throw new HttpError(response.status, `Request to ${path} failed with status ${response.status}`);
  }

  return (await response.json()) as T;
}

export async function httpFetchUsers(): Promise<User[]> {
  return request<User[]>("/api/users");
}

export async function httpCreateUser(payload: CreateUserRequest): Promise<User> {
  return request<User>("/api/users", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpInviteUser(payload: InviteUserRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/invite", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpAcceptInvite(payload: AcceptInviteRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/invite/accept", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpJoinRoom(payload: JoinRoomRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/join", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpLeaveRoom(payload: LeaveRoomRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/leave", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpFetchRooms(): Promise<Room[]> {
  return request<Room[]>("/api/rooms");
}

export async function httpCreateRoom(name: string): Promise<Room> {
  return request<Room>("/api/rooms", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export async function httpFetchChatHistory(roomId: string): Promise<ChatMessage[]> {
  return request<ChatMessage[]>(`/api/chat/${roomId}`);
}

export async function httpSendChatMessage(payload: ChatMessageRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/chat", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
