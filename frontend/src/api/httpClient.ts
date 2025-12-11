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
  Operation,
} from "./types";

let API_BASE_URL = "http://localhost:8080";

export function setApiBaseUrl(url: string) {
  // Ensure URL starts with http:// or https://
  if (!url.startsWith("http://") && !url.startsWith("https://")) {
    url = "http://" + url;
  }
  // Remove trailing slash if present
  if (url.endsWith("/")) {
    url = url.slice(0, -1);
  }
  // Add port 8080 only for local IPs without port
  // Don't add port for cloudflare tunnels or https URLs
  const hasPort = /:\d+$/.test(url);
  const isCloudflare = url.includes('trycloudflare.com') || url.includes('cloudflare');
  const isHttps = url.startsWith('https://');
  if (!hasPort && !isCloudflare && !isHttps) {
    url = url + ":8080";
  }
  
  API_BASE_URL = url;
  console.log("API Base URL set to:", API_BASE_URL);
}

export function getApiBaseUrl(): string {
  return API_BASE_URL;
}

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

export async function httpRequestJoin(payload: JoinRoomRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/join/request", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpApproveJoin(ownerId: string, requesterId: string, roomId: string): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/join/approve", {
    method: "POST",
    body: JSON.stringify({ ownerId, requesterId, roomId }),
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

export async function httpFetchOperations(roomId: string, sinceId: string = ""): Promise<Operation[]> {
  let url = `/api/operations/${roomId}`;
  if (sinceId) {
    url += `?since=${sinceId}`;
  }
  return request<Operation[]>(url);
}

